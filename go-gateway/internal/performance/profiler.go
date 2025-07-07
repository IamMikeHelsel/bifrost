package performance

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Profiler manages CPU and memory profiling capabilities
type Profiler struct {
	logger   *zap.Logger
	config   *ProfilerConfig
	server   *http.Server
	profiles map[string]*Profile
	mutex    sync.RWMutex

	// Active profiling sessions
	activeProfiles sync.Map

	// Metrics collection
	metrics *ProfilerMetrics

	// Control channels
	ctx    context.Context
	cancel context.CancelFunc
}

// ProfilerConfig defines profiling configuration
type ProfilerConfig struct {
	Enabled  bool   `yaml:"enabled"`
	HTTPPort int    `yaml:"http_port"`
	HTTPHost string `yaml:"http_host"`

	// Automatic profiling
	AutoCPUProfile       bool `yaml:"auto_cpu_profile"`
	AutoMemProfile       bool `yaml:"auto_mem_profile"`
	AutoGoroutineProfile bool `yaml:"auto_goroutine_profile"`
	AutoBlockProfile     bool `yaml:"auto_block_profile"`
	AutoMutexProfile     bool `yaml:"auto_mutex_profile"`

	// Profile collection intervals
	CPUProfileInterval       time.Duration `yaml:"cpu_profile_interval"`
	MemProfileInterval       time.Duration `yaml:"mem_profile_interval"`
	GoroutineProfileInterval time.Duration `yaml:"goroutine_profile_interval"`

	// Profile retention
	MaxProfiles      int           `yaml:"max_profiles"`
	ProfileRetention time.Duration `yaml:"profile_retention"`

	// Output configuration
	OutputDirectory  string `yaml:"output_directory"`
	CompressProfiles bool   `yaml:"compress_profiles"`

	// Performance thresholds for automatic profiling
	CPUThreshold       float64 `yaml:"cpu_threshold"`
	MemoryThreshold    int64   `yaml:"memory_threshold"`
	GoroutineThreshold int     `yaml:"goroutine_threshold"`
}

// ProfilerMetrics tracks profiling statistics
type ProfilerMetrics struct {
	ProfilesCollected  int64
	ProfilesCompressed int64
	ProfilesDeleted    int64

	// Profile sizes
	TotalProfileSize   int64
	AverageProfileSize int64
	LargestProfileSize int64

	// Performance impact
	ProfilingOverhead time.Duration
	CollectionLatency time.Duration

	// Resource usage during profiling
	CPUUsageDuringProfile    float64
	MemoryUsageDuringProfile int64
}

// Profile represents a collected profile
type Profile struct {
	ID         string
	Type       string
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Size       int64
	Compressed bool
	FilePath   string

	// Performance context
	CPUUsage    float64
	MemoryUsage int64
	Goroutines  int

	// Metadata
	Metadata map[string]interface{}
}

// NewProfiler creates a new profiler instance
func NewProfiler(config *ProfilerConfig, logger *zap.Logger) *Profiler {
	ctx, cancel := context.WithCancel(context.Background())

	profiler := &Profiler{
		logger:   logger,
		config:   config,
		profiles: make(map[string]*Profile),
		metrics:  &ProfilerMetrics{},
		ctx:      ctx,
		cancel:   cancel,
	}

	if config.Enabled {
		profiler.initialize()
	}

	return profiler
}

// initialize sets up the profiler
func (p *Profiler) initialize() {
	// Create output directory
	if err := os.MkdirAll(p.config.OutputDirectory, 0755); err != nil {
		p.logger.Error("Failed to create profile output directory", zap.Error(err))
		return
	}

	// Start HTTP server for pprof endpoints
	if p.config.HTTPPort > 0 {
		go p.startHTTPServer()
	}

	// Start automatic profiling
	if p.config.AutoCPUProfile {
		go p.startAutomaticCPUProfiling()
	}

	if p.config.AutoMemProfile {
		go p.startAutomaticMemProfiling()
	}

	if p.config.AutoGoroutineProfile {
		go p.startAutomaticGoroutineProfiling()
	}

	// Start profile cleanup
	go p.startProfileCleanup()

	p.logger.Info("Profiler initialized",
		zap.String("output_dir", p.config.OutputDirectory),
		zap.Int("http_port", p.config.HTTPPort),
		zap.Bool("auto_cpu", p.config.AutoCPUProfile),
		zap.Bool("auto_mem", p.config.AutoMemProfile),
	)
}

// startHTTPServer starts the HTTP server for pprof endpoints
func (p *Profiler) startHTTPServer() {
	mux := http.NewServeMux()

	// Add pprof endpoints
	mux.HandleFunc("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.HandleFunc("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.HandleFunc("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.HandleFunc("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.HandleFunc("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	// Add custom endpoints
	mux.HandleFunc("/debug/pprof/heap", p.handleHeapProfile)
	mux.HandleFunc("/debug/pprof/goroutine", p.handleGoroutineProfile)
	mux.HandleFunc("/debug/pprof/block", p.handleBlockProfile)
	mux.HandleFunc("/debug/pprof/mutex", p.handleMutexProfile)
	mux.HandleFunc("/debug/profiles", p.handleProfileList)
	mux.HandleFunc("/debug/profiles/", p.handleProfileDownload)

	p.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", p.config.HTTPHost, p.config.HTTPPort),
		Handler: mux,
	}

	p.logger.Info("Profile HTTP server starting",
		zap.String("addr", p.server.Addr),
	)

	if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		p.logger.Error("Profile HTTP server error", zap.Error(err))
	}
}

// startAutomaticCPUProfiling starts automatic CPU profiling
func (p *Profiler) startAutomaticCPUProfiling() {
	ticker := time.NewTicker(p.config.CPUProfileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if p.shouldCollectCPUProfile() {
				p.CollectCPUProfile(30 * time.Second)
			}
		}
	}
}

// startAutomaticMemProfiling starts automatic memory profiling
func (p *Profiler) startAutomaticMemProfiling() {
	ticker := time.NewTicker(p.config.MemProfileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if p.shouldCollectMemProfile() {
				p.CollectMemProfile()
			}
		}
	}
}

// startAutomaticGoroutineProfiling starts automatic goroutine profiling
func (p *Profiler) startAutomaticGoroutineProfiling() {
	ticker := time.NewTicker(p.config.GoroutineProfileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if p.shouldCollectGoroutineProfile() {
				p.CollectGoroutineProfile()
			}
		}
	}
}

// shouldCollectCPUProfile determines if CPU profiling should be triggered
func (p *Profiler) shouldCollectCPUProfile() bool {
	// Check if CPU usage exceeds threshold
	cpuUsage := p.getCurrentCPUUsage()
	return cpuUsage > p.config.CPUThreshold
}

// shouldCollectMemProfile determines if memory profiling should be triggered
func (p *Profiler) shouldCollectMemProfile() bool {
	// Check if memory usage exceeds threshold
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.HeapAlloc) > p.config.MemoryThreshold
}

// shouldCollectGoroutineProfile determines if goroutine profiling should be triggered
func (p *Profiler) shouldCollectGoroutineProfile() bool {
	// Check if goroutine count exceeds threshold
	goroutines := runtime.NumGoroutine()
	return goroutines > p.config.GoroutineThreshold
}

// CollectCPUProfile collects a CPU profile
func (p *Profiler) CollectCPUProfile(duration time.Duration) (*Profile, error) {
	start := time.Now()

	profile := &Profile{
		ID:        fmt.Sprintf("cpu_%d", time.Now().Unix()),
		Type:      "cpu",
		StartTime: start,
		Duration:  duration,
		Metadata:  make(map[string]interface{}),
	}

	filename := fmt.Sprintf("%s/%s.prof", p.config.OutputDirectory, profile.ID)
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create CPU profile file: %w", err)
	}
	defer file.Close()

	// Start CPU profiling
	if err := pprof.StartCPUProfile(file); err != nil {
		return nil, fmt.Errorf("failed to start CPU profile: %w", err)
	}

	// Store active profile
	p.activeProfiles.Store(profile.ID, profile)

	// Stop profiling after duration
	go func() {
		time.Sleep(duration)
		pprof.StopCPUProfile()

		// Finalize profile
		profile.EndTime = time.Now()
		profile.Duration = profile.EndTime.Sub(profile.StartTime)
		profile.FilePath = filename

		// Get file size
		if info, err := os.Stat(filename); err == nil {
			profile.Size = info.Size()
		}

		// Add performance context
		profile.CPUUsage = p.getCurrentCPUUsage()
		profile.MemoryUsage = p.getCurrentMemoryUsage()
		profile.Goroutines = runtime.NumGoroutine()

		// Compress if enabled
		if p.config.CompressProfiles {
			if err := p.compressProfile(profile); err != nil {
				p.logger.Error("Failed to compress profile", zap.Error(err))
			}
		}

		// Store profile
		p.storeProfile(profile)

		// Remove from active profiles
		p.activeProfiles.Delete(profile.ID)

		p.logger.Info("CPU profile collected",
			zap.String("profile_id", profile.ID),
			zap.Duration("duration", profile.Duration),
			zap.Int64("size", profile.Size),
		)
	}()

	return profile, nil
}

// CollectMemProfile collects a memory profile
func (p *Profiler) CollectMemProfile() (*Profile, error) {
	start := time.Now()

	profile := &Profile{
		ID:        fmt.Sprintf("mem_%d", time.Now().Unix()),
		Type:      "memory",
		StartTime: start,
		Metadata:  make(map[string]interface{}),
	}

	filename := fmt.Sprintf("%s/%s.prof", p.config.OutputDirectory, profile.ID)
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory profile file: %w", err)
	}
	defer file.Close()

	// Force GC to get accurate memory profile
	runtime.GC()

	// Write memory profile
	if err := pprof.WriteHeapProfile(file); err != nil {
		return nil, fmt.Errorf("failed to write memory profile: %w", err)
	}

	// Finalize profile
	profile.EndTime = time.Now()
	profile.Duration = profile.EndTime.Sub(profile.StartTime)
	profile.FilePath = filename

	// Get file size
	if info, err := os.Stat(filename); err == nil {
		profile.Size = info.Size()
	}

	// Add performance context
	profile.CPUUsage = p.getCurrentCPUUsage()
	profile.MemoryUsage = p.getCurrentMemoryUsage()
	profile.Goroutines = runtime.NumGoroutine()

	// Compress if enabled
	if p.config.CompressProfiles {
		if err := p.compressProfile(profile); err != nil {
			p.logger.Error("Failed to compress profile", zap.Error(err))
		}
	}

	// Store profile
	p.storeProfile(profile)

	p.logger.Info("Memory profile collected",
		zap.String("profile_id", profile.ID),
		zap.Int64("size", profile.Size),
		zap.Int64("memory_usage", profile.MemoryUsage),
	)

	return profile, nil
}

// CollectGoroutineProfile collects a goroutine profile
func (p *Profiler) CollectGoroutineProfile() (*Profile, error) {
	start := time.Now()

	profile := &Profile{
		ID:        fmt.Sprintf("goroutine_%d", time.Now().Unix()),
		Type:      "goroutine",
		StartTime: start,
		Metadata:  make(map[string]interface{}),
	}

	filename := fmt.Sprintf("%s/%s.prof", p.config.OutputDirectory, profile.ID)
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create goroutine profile file: %w", err)
	}
	defer file.Close()

	// Write goroutine profile
	if err := pprof.Lookup("goroutine").WriteTo(file, 0); err != nil {
		return nil, fmt.Errorf("failed to write goroutine profile: %w", err)
	}

	// Finalize profile
	profile.EndTime = time.Now()
	profile.Duration = profile.EndTime.Sub(profile.StartTime)
	profile.FilePath = filename

	// Get file size
	if info, err := os.Stat(filename); err == nil {
		profile.Size = info.Size()
	}

	// Add performance context
	profile.CPUUsage = p.getCurrentCPUUsage()
	profile.MemoryUsage = p.getCurrentMemoryUsage()
	profile.Goroutines = runtime.NumGoroutine()

	// Compress if enabled
	if p.config.CompressProfiles {
		if err := p.compressProfile(profile); err != nil {
			p.logger.Error("Failed to compress profile", zap.Error(err))
		}
	}

	// Store profile
	p.storeProfile(profile)

	p.logger.Info("Goroutine profile collected",
		zap.String("profile_id", profile.ID),
		zap.Int64("size", profile.Size),
		zap.Int("goroutines", profile.Goroutines),
	)

	return profile, nil
}

// StartTrace starts execution tracing
func (p *Profiler) StartTrace(duration time.Duration) (*Profile, error) {
	start := time.Now()

	profile := &Profile{
		ID:        fmt.Sprintf("trace_%d", time.Now().Unix()),
		Type:      "trace",
		StartTime: start,
		Duration:  duration,
		Metadata:  make(map[string]interface{}),
	}

	filename := fmt.Sprintf("%s/%s.trace", p.config.OutputDirectory, profile.ID)
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace file: %w", err)
	}

	// Start tracing
	if err := trace.Start(file); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to start trace: %w", err)
	}

	// Store active profile
	p.activeProfiles.Store(profile.ID, profile)

	// Stop tracing after duration
	go func() {
		time.Sleep(duration)
		trace.Stop()
		file.Close()

		// Finalize profile
		profile.EndTime = time.Now()
		profile.Duration = profile.EndTime.Sub(profile.StartTime)
		profile.FilePath = filename

		// Get file size
		if info, err := os.Stat(filename); err == nil {
			profile.Size = info.Size()
		}

		// Add performance context
		profile.CPUUsage = p.getCurrentCPUUsage()
		profile.MemoryUsage = p.getCurrentMemoryUsage()
		profile.Goroutines = runtime.NumGoroutine()

		// Compress if enabled
		if p.config.CompressProfiles {
			if err := p.compressProfile(profile); err != nil {
				p.logger.Error("Failed to compress profile", zap.Error(err))
			}
		}

		// Store profile
		p.storeProfile(profile)

		// Remove from active profiles
		p.activeProfiles.Delete(profile.ID)

		p.logger.Info("Trace collected",
			zap.String("profile_id", profile.ID),
			zap.Duration("duration", profile.Duration),
			zap.Int64("size", profile.Size),
		)
	}()

	return profile, nil
}

// storeProfile stores a profile in the registry
func (p *Profiler) storeProfile(profile *Profile) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.profiles[profile.ID] = profile
	p.metrics.ProfilesCollected++

	// Update metrics
	p.metrics.TotalProfileSize += profile.Size
	p.metrics.AverageProfileSize = p.metrics.TotalProfileSize / p.metrics.ProfilesCollected

	if profile.Size > p.metrics.LargestProfileSize {
		p.metrics.LargestProfileSize = profile.Size
	}

	if profile.Compressed {
		p.metrics.ProfilesCompressed++
	}

	// Clean up old profiles if needed
	if len(p.profiles) > p.config.MaxProfiles {
		p.cleanupOldProfiles()
	}
}

// Helper functions

func (p *Profiler) getCurrentCPUUsage() float64 {
	// This would implement actual CPU usage calculation
	// For now, return a mock value
	return 45.0
}

func (p *Profiler) getCurrentMemoryUsage() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.HeapAlloc)
}

func (p *Profiler) compressProfile(profile *Profile) error {
	// This would implement profile compression
	// For now, just mark as compressed
	profile.Compressed = true
	return nil
}

func (p *Profiler) cleanupOldProfiles() {
	// Remove oldest profiles
	// This would implement proper cleanup logic
}

func (p *Profiler) startProfileCleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.cleanupExpiredProfiles()
		}
	}
}

func (p *Profiler) cleanupExpiredProfiles() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	cutoff := time.Now().Add(-p.config.ProfileRetention)

	for id, profile := range p.profiles {
		if profile.StartTime.Before(cutoff) {
			// Remove expired profile
			if err := os.Remove(profile.FilePath); err != nil {
				p.logger.Error("Failed to remove expired profile", zap.Error(err))
			}
			delete(p.profiles, id)
			p.metrics.ProfilesDeleted++
		}
	}
}

// HTTP handlers

func (p *Profiler) handleHeapProfile(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("heap").ServeHTTP(w, r)
}

func (p *Profiler) handleGoroutineProfile(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("goroutine").ServeHTTP(w, r)
}

func (p *Profiler) handleBlockProfile(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("block").ServeHTTP(w, r)
}

func (p *Profiler) handleMutexProfile(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("mutex").ServeHTTP(w, r)
}

func (p *Profiler) handleProfileList(w http.ResponseWriter, r *http.Request) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	// Return JSON list of profiles
	fmt.Fprintf(w, `{"profiles": %d}`, len(p.profiles))
}

func (p *Profiler) handleProfileDownload(w http.ResponseWriter, r *http.Request) {
	// Handle profile download
	profileID := r.URL.Path[len("/debug/profiles/"):]

	p.mutex.RLock()
	profile, exists := p.profiles[profileID]
	p.mutex.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, profile.FilePath)
}

// GetMetrics returns profiler metrics
func (p *Profiler) GetMetrics() *ProfilerMetrics {
	return p.metrics
}

// GetProfiles returns all collected profiles
func (p *Profiler) GetProfiles() map[string]*Profile {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Return a copy to prevent concurrent access issues
	profiles := make(map[string]*Profile)
	for k, v := range p.profiles {
		profiles[k] = v
	}
	return profiles
}

// Close gracefully shuts down the profiler
func (p *Profiler) Close() error {
	p.cancel()

	// Stop any active profiling
	p.activeProfiles.Range(func(key, value interface{}) bool {
		profile := value.(*Profile)
		if profile.Type == "cpu" {
			pprof.StopCPUProfile()
		} else if profile.Type == "trace" {
			trace.Stop()
		}
		return true
	})

	// Stop HTTP server
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p.server.Shutdown(ctx)
	}

	return nil
}
