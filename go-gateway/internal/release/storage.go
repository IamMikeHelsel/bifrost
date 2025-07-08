package release

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileStorageProvider implements StorageProvider using local filesystem
type FileStorageProvider struct {
	baseDir string
}

// NewFileStorageProvider creates a new file-based storage provider
func NewFileStorageProvider(baseDir string) *FileStorageProvider {
	return &FileStorageProvider{
		baseDir: baseDir,
	}
}

// Store stores a release card in the specified format
func (fsp *FileStorageProvider) Store(card *ReleaseCard, format string) error {
	// Ensure directory exists
	if err := os.MkdirAll(fsp.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	filename := fsp.getFilename(card.Metadata.Version, format)
	filepath := filepath.Join(fsp.baseDir, filename)
	
	var data []byte
	var err error
	
	switch format {
	case "yaml", "yml":
		data, err = card.ToYAML()
	case "json":
		data, err = card.ToJSON()
	default:
		return fmt.Errorf("unsupported storage format: %s", format)
	}
	
	if err != nil {
		return fmt.Errorf("failed to serialize card: %w", err)
	}
	
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// Load loads a release card by version
func (fsp *FileStorageProvider) Load(version string) (*ReleaseCard, error) {
	// Try to load from YAML first, then JSON
	for _, ext := range []string{"yaml", "json"} {
		filename := fsp.getFilename(version, ext)
		filepath := filepath.Join(fsp.baseDir, filename)
		
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			continue
		}
		
		data, err := os.ReadFile(filepath)
		if err != nil {
			continue
		}
		
		card := &ReleaseCard{}
		switch ext {
		case "yaml":
			err = card.FromYAML(data)
		case "json":
			err = card.FromJSON(data)
		}
		
		if err == nil {
			return card, nil
		}
	}
	
	return nil, fmt.Errorf("release card not found for version: %s", version)
}

// List loads all available release cards
func (fsp *FileStorageProvider) List() ([]*ReleaseCard, error) {
	files, err := os.ReadDir(fsp.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ReleaseCard{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	
	cards := make([]*ReleaseCard, 0)
	versions := make(map[string]bool)
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		name := file.Name()
		if !strings.HasPrefix(name, "release-card-") {
			continue
		}
		
		// Extract version from filename
		version, _ := fsp.parseFilename(name)
		if version == "" || versions[version] {
			continue // Skip if already processed this version
		}
		
		card, err := fsp.Load(version)
		if err != nil {
			continue // Skip files that can't be loaded
		}
		
		cards = append(cards, card)
		versions[version] = true
	}
	
	// Sort by release date
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Metadata.ReleaseDate.Before(cards[j].Metadata.ReleaseDate)
	})
	
	return cards, nil
}

// Delete removes a release card by version
func (fsp *FileStorageProvider) Delete(version string) error {
	// Delete all formats
	for _, format := range []string{"yaml", "json"} {
		filename := fsp.getFilename(version, format)
		filepath := filepath.Join(fsp.baseDir, filename)
		
		if _, err := os.Stat(filepath); err == nil {
			if err := os.Remove(filepath); err != nil {
				return fmt.Errorf("failed to delete file %s: %w", filepath, err)
			}
		}
	}
	
	return nil
}

// getFilename generates a filename for a version and format
func (fsp *FileStorageProvider) getFilename(version, format string) string {
	return fmt.Sprintf("release-card-%s.%s", version, format)
}

// parseFilename extracts version and format from a filename
func (fsp *FileStorageProvider) parseFilename(filename string) (version, format string) {
	if !strings.HasPrefix(filename, "release-card-") {
		return "", ""
	}
	
	// Remove prefix
	remainder := strings.TrimPrefix(filename, "release-card-")
	
	// Find last dot
	lastDot := strings.LastIndex(remainder, ".")
	if lastDot == -1 {
		return "", ""
	}
	
	version = remainder[:lastDot]
	format = remainder[lastDot+1:]
	
	return version, format
}

// InMemoryStorageProvider implements StorageProvider using in-memory storage (for testing)
type InMemoryStorageProvider struct {
	cards map[string]*ReleaseCard
}

// NewInMemoryStorageProvider creates a new in-memory storage provider
func NewInMemoryStorageProvider() *InMemoryStorageProvider {
	return &InMemoryStorageProvider{
		cards: make(map[string]*ReleaseCard),
	}
}

// Store stores a release card in memory
func (isp *InMemoryStorageProvider) Store(card *ReleaseCard, format string) error {
	// Clone the card to avoid reference issues
	cardData, err := json.Marshal(card)
	if err != nil {
		return err
	}
	
	cardCopy := &ReleaseCard{}
	if err := json.Unmarshal(cardData, cardCopy); err != nil {
		return err
	}
	
	isp.cards[card.Metadata.Version] = cardCopy
	return nil
}

// Load loads a release card from memory
func (isp *InMemoryStorageProvider) Load(version string) (*ReleaseCard, error) {
	card, exists := isp.cards[version]
	if !exists {
		return nil, fmt.Errorf("release card not found for version: %s", version)
	}
	
	// Clone the card to avoid reference issues
	cardData, err := json.Marshal(card)
	if err != nil {
		return nil, err
	}
	
	cardCopy := &ReleaseCard{}
	if err := json.Unmarshal(cardData, cardCopy); err != nil {
		return nil, err
	}
	
	return cardCopy, nil
}

// List returns all stored release cards
func (isp *InMemoryStorageProvider) List() ([]*ReleaseCard, error) {
	cards := make([]*ReleaseCard, 0, len(isp.cards))
	
	for _, card := range isp.cards {
		// Clone the card to avoid reference issues
		cardData, err := json.Marshal(card)
		if err != nil {
			continue
		}
		
		cardCopy := &ReleaseCard{}
		if err := json.Unmarshal(cardData, cardCopy); err != nil {
			continue
		}
		
		cards = append(cards, cardCopy)
	}
	
	// Sort by release date
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Metadata.ReleaseDate.Before(cards[j].Metadata.ReleaseDate)
	})
	
	return cards, nil
}

// Delete removes a release card from memory
func (isp *InMemoryStorageProvider) Delete(version string) error {
	delete(isp.cards, version)
	return nil
}

// Clear removes all stored cards (for testing)
func (isp *InMemoryStorageProvider) Clear() {
	isp.cards = make(map[string]*ReleaseCard)
}