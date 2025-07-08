package testframework

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"testing"
	"time"

	"go.uber.org/zap"
)

// FuzzTest represents a single fuzzing test configuration
type FuzzTest struct {
	Name            string
	Target          string // Protocol or component being fuzzed
	InputGenerator  func() []byte
	ResponseChecker func([]byte) error
	MaxIterations   int
	TimeLimit       time.Duration
	CrashDetector   func(error) bool
}

// FuzzResult contains the results of a fuzzing test
type FuzzResult struct {
	TestName        string
	TotalInputs     int
	CrashingInputs  [][]byte
	ErrorCount      int
	UniqueErrors    map[string]int
	Duration        time.Duration
	CoveragePercent float64
	Success         bool
}

// FuzzSuite manages a collection of fuzzing tests
type FuzzSuite struct {
	logger *zap.Logger
	tests  []FuzzTest
}

// NewFuzzSuite creates a new fuzzing test suite
func NewFuzzSuite(logger *zap.Logger) *FuzzSuite {
	return &FuzzSuite{
		logger: logger,
		tests:  make([]FuzzTest, 0),
	}
}

// AddTest adds a fuzzing test to the suite
func (fs *FuzzSuite) AddTest(test FuzzTest) {
	fs.tests = append(fs.tests, test)
}

// RunAll runs all fuzzing tests in the suite
func (fs *FuzzSuite) RunAll(ctx context.Context) ([]FuzzResult, error) {
	results := make([]FuzzResult, 0, len(fs.tests))

	for _, test := range fs.tests {
		fs.logger.Info("Starting fuzz test", zap.String("test", test.Name))
		
		result, err := fs.runSingleTest(ctx, test)
		if err != nil {
			fs.logger.Error("Fuzz test failed", zap.String("test", test.Name), zap.Error(err))
			result.Success = false
		}
		
		results = append(results, result)
		
		fs.logger.Info("Fuzz test completed", 
			zap.String("test", test.Name),
			zap.Int("inputs_tested", result.TotalInputs),
			zap.Int("crashes_found", len(result.CrashingInputs)),
			zap.Bool("success", result.Success),
		)
	}

	return results, nil
}

// runSingleTest executes a single fuzzing test
func (fs *FuzzSuite) runSingleTest(ctx context.Context, test FuzzTest) (FuzzResult, error) {
	result := FuzzResult{
		TestName:       test.Name,
		UniqueErrors:   make(map[string]int),
		CrashingInputs: make([][]byte, 0),
	}

	startTime := time.Now()
	deadline := startTime.Add(test.TimeLimit)

	for i := 0; i < test.MaxIterations && time.Now().Before(deadline); i++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Generate test input
		input := test.InputGenerator()
		result.TotalInputs++

		// Test the input and check for crashes/errors
		if err := fs.testInput(input, test); err != nil {
			result.ErrorCount++
			
			errorKey := err.Error()
			result.UniqueErrors[errorKey]++

			// Check if this is a crash
			if test.CrashDetector != nil && test.CrashDetector(err) {
				// Store the crashing input (up to a reasonable limit)
				if len(result.CrashingInputs) < 100 {
					inputCopy := make([]byte, len(input))
					copy(inputCopy, input)
					result.CrashingInputs = append(result.CrashingInputs, inputCopy)
				}
			}
		}
	}

	result.Duration = time.Since(startTime)
	result.Success = len(result.CrashingInputs) == 0
	result.CoveragePercent = fs.calculateCoverage(result.TotalInputs)

	return result, nil
}

// testInput tests a single input against the target
func (fs *FuzzSuite) testInput(input []byte, test FuzzTest) error {
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to error
			panic(fmt.Errorf("panic during fuzzing: %v", r))
		}
	}()

	// In a real implementation, this would send the input to the target protocol
	// For now, we'll simulate testing by checking for known bad patterns
	return fs.simulateProtocolTest(input, test.Target)
}

// simulateProtocolTest simulates testing input against a protocol
func (fs *FuzzSuite) simulateProtocolTest(input []byte, target string) error {
	switch target {
	case "modbus":
		return fs.testModbusInput(input)
	case "ethernetip":
		return fs.testEtherNetIPInput(input)
	case "config":
		return fs.testConfigInput(input)
	default:
		return nil
	}
}

// testModbusInput tests Modbus protocol input
func (fs *FuzzSuite) testModbusInput(input []byte) error {
	// Simulate Modbus protocol validation
	if len(input) < 8 {
		return fmt.Errorf("modbus packet too short")
	}

	// Check for malformed transaction ID
	if len(input) >= 2 && input[0] == 0xFF && input[1] == 0xFF {
		return fmt.Errorf("invalid transaction ID")
	}

	// Check for invalid function codes
	if len(input) >= 8 && input[7] > 0x7F {
		return fmt.Errorf("invalid function code: %d", input[7])
	}

	// Simulate potential buffer overflow
	if len(input) > 256 {
		return fmt.Errorf("potential buffer overflow - packet too large")
	}

	return nil
}

// testEtherNetIPInput tests EtherNet/IP protocol input
func (fs *FuzzSuite) testEtherNetIPInput(input []byte) error {
	if len(input) < 24 {
		return fmt.Errorf("ethernetip packet too short")
	}

	// Check for invalid command codes
	if len(input) >= 4 && (input[0] > 0x70 || input[0] == 0x00) {
		return fmt.Errorf("invalid command code: %d", input[0])
	}

	// Check for session handle issues
	if len(input) >= 8 && bytes.Equal(input[4:8], []byte{0xFF, 0xFF, 0xFF, 0xFF}) {
		return fmt.Errorf("invalid session handle")
	}

	return nil
}

// testConfigInput tests configuration file input
func (fs *FuzzSuite) testConfigInput(input []byte) error {
	// Look for potential injection patterns
	dangerousPatterns := []string{
		"../", "\\..\\", "/etc/passwd", "cmd.exe",
		"<script>", "javascript:", "data:text/html",
	}

	inputStr := string(input)
	for _, pattern := range dangerousPatterns {
		if bytes.Contains(input, []byte(pattern)) {
			return fmt.Errorf("dangerous pattern detected: %s", pattern)
		}
	}

	// Check for extremely long lines that could cause buffer overflows
	lines := bytes.Split(input, []byte("\n"))
	for _, line := range lines {
		if len(line) > 10000 {
			return fmt.Errorf("line too long: %d characters", len(line))
		}
	}

	return nil
}

// calculateCoverage estimates code coverage based on inputs tested
func (fs *FuzzSuite) calculateCoverage(inputCount int) float64 {
	// Simplified coverage calculation
	// In a real implementation, this would integrate with Go's coverage tools
	maxCoverage := 95.0
	return math.Min(maxCoverage, float64(inputCount)/1000.0*maxCoverage)
}

// Input generators for different types of fuzzing

// RandomBytesGenerator generates random byte arrays
func RandomBytesGenerator(minLen, maxLen int) func() []byte {
	return func() []byte {
		length := minLen + int(randomUint32())%(maxLen-minLen+1)
		data := make([]byte, length)
		rand.Read(data)
		return data
	}
}

// ModbusFuzzGenerator generates Modbus-like packets with mutations
func ModbusFuzzGenerator() func() []byte {
	return func() []byte {
		// Base Modbus TCP packet structure
		packet := make([]byte, 8+int(randomUint32()%248)) // Variable length payload
		
		// Transaction ID (2 bytes)
		packet[0] = byte(randomUint32())
		packet[1] = byte(randomUint32())
		
		// Protocol ID (2 bytes) - usually 0x0000 for Modbus TCP
		if randomUint32()%10 == 0 { // 10% chance to corrupt
			packet[2] = byte(randomUint32())
			packet[3] = byte(randomUint32())
		} else {
			packet[2] = 0x00
			packet[3] = 0x00
		}
		
		// Length (2 bytes)
		length := len(packet) - 6
		packet[4] = byte(length >> 8)
		packet[5] = byte(length)
		
		// Unit ID (1 byte)
		packet[6] = byte(randomUint32() % 256)
		
		// Function code (1 byte)
		functionCodes := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x0F, 0x10}
		if randomUint32()%20 == 0 { // 5% chance for invalid function code
			packet[7] = byte(randomUint32())
		} else {
			packet[7] = functionCodes[randomUint32()%uint32(len(functionCodes))]
		}
		
		// Fill remaining bytes with random data
		for i := 8; i < len(packet); i++ {
			packet[i] = byte(randomUint32())
		}
		
		return packet
	}
}

// EtherNetIPFuzzGenerator generates EtherNet/IP-like packets
func EtherNetIPFuzzGenerator() func() []byte {
	return func() []byte {
		// Base EtherNet/IP encapsulation header
		packet := make([]byte, 24+int(randomUint32()%1000))
		
		// Command (2 bytes)
		commands := []uint16{0x0065, 0x0066, 0x006F, 0x0070}
		if randomUint32()%20 == 0 { // 5% chance for invalid command
			packet[0] = byte(randomUint32())
			packet[1] = byte(randomUint32())
		} else {
			cmd := commands[randomUint32()%uint32(len(commands))]
			packet[0] = byte(cmd)
			packet[1] = byte(cmd >> 8)
		}
		
		// Length (2 bytes)
		length := len(packet) - 24
		packet[2] = byte(length)
		packet[3] = byte(length >> 8)
		
		// Session handle (4 bytes)
		if randomUint32()%10 == 0 { // 10% chance for invalid handle
			for i := 4; i < 8; i++ {
				packet[i] = 0xFF
			}
		} else {
			for i := 4; i < 8; i++ {
				packet[i] = byte(randomUint32())
			}
		}
		
		// Fill remaining bytes
		for i := 8; i < len(packet); i++ {
			packet[i] = byte(randomUint32())
		}
		
		return packet
	}
}

// ConfigFuzzGenerator generates configuration-like text with mutations
func ConfigFuzzGenerator() func() []byte {
	return func() []byte {
		templates := []string{
			"gateway:\n  port: %d\n  protocols:\n    modbus:\n      enabled: %s",
			"devices:\n  - name: \"%s\"\n    address: \"%s\"\n    tags:\n      - name: \"%s\"",
			"logging:\n  level: \"%s\"\n  file: \"%s\"\n  max_size: %d",
		}
		
		template := templates[randomUint32()%uint32(len(templates))]
		
		// Generate potentially problematic values
		var config string
		switch randomUint32() % 4 {
		case 0: // Normal values
			config = fmt.Sprintf(template, 502, "true", "device1", "192.168.1.1", "tag1", "info", "app.log", 100)
		case 1: // Very long strings
			longStr := string(bytes.Repeat([]byte("A"), 10000))
			config = fmt.Sprintf(template, 65536, longStr, longStr, longStr, longStr, longStr, longStr, 999999)
		case 2: // Special characters and injection attempts
			config = fmt.Sprintf(template, -1, "../../../etc/passwd", "<script>alert('xss')</script>", 
				"'; DROP TABLE devices; --", "$(rm -rf /)", "javascript:alert(1)", "../../../../windows/system32/cmd.exe", 0)
		case 3: // Malformed structure
			config = "}}}}invalid yaml{{{{ with \x00\x01\x02 binary data"
		}
		
		return []byte(config)
	}
}

// randomUint32 generates a random uint32 value
func randomUint32() uint32 {
	var b [4]byte
	rand.Read(b[:])
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// CreateStandardFuzzTests creates a standard set of fuzzing tests
func CreateStandardFuzzTests() []FuzzTest {
	return []FuzzTest{
		{
			Name:           "Modbus Protocol Fuzzing",
			Target:         "modbus",
			InputGenerator: ModbusFuzzGenerator(),
			MaxIterations:  10000,
			TimeLimit:      5 * time.Minute,
			CrashDetector: func(err error) bool {
				// Consider these as crashes
				crashKeywords := []string{"panic", "overflow", "crash", "segmentation", "buffer"}
				errStr := err.Error()
				for _, keyword := range crashKeywords {
					if bytes.Contains([]byte(errStr), []byte(keyword)) {
						return true
					}
				}
				return false
			},
		},
		{
			Name:           "EtherNet/IP Protocol Fuzzing",
			Target:         "ethernetip",
			InputGenerator: EtherNetIPFuzzGenerator(),
			MaxIterations:  10000,
			TimeLimit:      5 * time.Minute,
			CrashDetector: func(err error) bool {
				crashKeywords := []string{"panic", "overflow", "crash"}
				errStr := err.Error()
				for _, keyword := range crashKeywords {
					if bytes.Contains([]byte(errStr), []byte(keyword)) {
						return true
					}
				}
				return false
			},
		},
		{
			Name:           "Configuration Fuzzing",
			Target:         "config",
			InputGenerator: ConfigFuzzGenerator(),
			MaxIterations:  5000,
			TimeLimit:      3 * time.Minute,
			CrashDetector: func(err error) bool {
				// Configuration errors are usually not crashes, but we want to catch injection attempts
				dangerousErrors := []string{"injection", "overflow", "path traversal"}
				errStr := err.Error()
				for _, danger := range dangerousErrors {
					if bytes.Contains([]byte(errStr), []byte(danger)) {
						return true
					}
				}
				return false
			},
		},
		{
			Name:           "Random Binary Fuzzing",
			Target:         "general",
			InputGenerator: RandomBytesGenerator(1, 2048),
			MaxIterations:  15000,
			TimeLimit:      10 * time.Minute,
			CrashDetector: func(err error) bool {
				return err != nil // Any error is considered significant for random input
			},
		},
	}
}

// RunFuzzTestSuite is a helper function for running fuzzing tests in Go test files
func RunFuzzTestSuite(t *testing.T, logger *zap.Logger) {
	suite := NewFuzzSuite(logger)
	
	// Add standard tests
	for _, test := range CreateStandardFuzzTests() {
		suite.AddTest(test)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	
	results, err := suite.RunAll(ctx)
	if err != nil {
		t.Fatalf("Fuzz suite failed: %v", err)
	}
	
	// Check results
	for _, result := range results {
		t.Logf("Fuzz test %s: %d inputs, %d crashes, %d errors, %.1f%% coverage",
			result.TestName, result.TotalInputs, len(result.CrashingInputs), 
			result.ErrorCount, result.CoveragePercent)
		
		// Fail if we found crashes (in a real implementation, this might be configurable)
		if len(result.CrashingInputs) > 0 {
			t.Errorf("Fuzz test %s found %d crashing inputs", result.TestName, len(result.CrashingInputs))
		}
	}
}