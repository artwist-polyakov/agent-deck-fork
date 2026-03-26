package experiments

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sahilm/fuzzy"
)

// Experiment represents a discovered or created experiment folder
type Experiment struct {
	Name    string    // Display name (without date prefix)
	Path    string    // Full path to directory
	Date    time.Time // Parsed date from folder name (if any)
	HasDate bool      // Whether folder has date prefix
	ModTime time.Time // Last modification time
}

// ListExperiments returns all experiment folders in the directory
// Sorted by modification time (most recent first)
func ListExperiments(dir string) ([]Experiment, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Experiment{}, nil
		}
		return nil, err
	}

	var experiments []Experiment
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		exp := Experiment{
			Name: name,
			Path: filepath.Join(dir, name),
		}

		// Try to parse date from folder name (YYYY-MM-DD-rest)
		if len(name) >= 11 && name[4] == '-' && name[7] == '-' && name[10] == '-' {
			if t, err := time.Parse("2006-01-02", name[:10]); err == nil {
				exp.Date = t
				exp.HasDate = true
				exp.Name = name[11:] // Strip date prefix for display
			}
		}

		// Get modification time
		if info, err := entry.Info(); err == nil {
			exp.ModTime = info.ModTime()
		}

		experiments = append(experiments, exp)
	}

	// Sort by modification time (most recent first)
	sort.Slice(experiments, func(i, j int) bool {
		return experiments[i].ModTime.After(experiments[j].ModTime)
	})

	return experiments, nil
}

// fuzzySource implements fuzzy.Source for experiments
type fuzzySource struct {
	experiments []Experiment
}

func (s fuzzySource) String(i int) string {
	return s.experiments[i].Name
}

func (s fuzzySource) Len() int {
	return len(s.experiments)
}

// FuzzyFind returns experiments matching the query, sorted by relevance
func FuzzyFind(experiments []Experiment, query string) []Experiment {
	if query == "" {
		return experiments
	}

	source := fuzzySource{experiments: experiments}
	matches := fuzzy.FindFrom(query, source)

	results := make([]Experiment, 0, len(matches))
	for _, match := range matches {
		results = append(results, experiments[match.Index])
	}
	return results
}

// FindExact returns an experiment with exact name match (case-insensitive)
func FindExact(experiments []Experiment, name string) *Experiment {
	nameLower := strings.ToLower(name)
	for i := range experiments {
		if strings.ToLower(experiments[i].Name) == nameLower {
			return &experiments[i] // Safe: points to slice element
		}
	}
	return nil
}

// CreateExperiment creates a new experiment folder
// If datePrefix is true, prepends YYYY-MM-DD- to the name
// Returns the created experiment
func CreateExperiment(baseDir, name string, datePrefix bool) (*Experiment, error) {
	// Sanitize name (replace spaces with hyphens, lowercase)
	name = strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	// Build folder name
	folderName := name
	if datePrefix {
		today := time.Now().Format("2006-01-02")
		folderName = today + "-" + name
	}

	// Handle duplicates
	targetPath := filepath.Join(baseDir, folderName)
	suffix := 2
	for {
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			break
		}
		// Path exists, try with suffix
		if datePrefix {
			today := time.Now().Format("2006-01-02")
			folderName = fmt.Sprintf("%s-%s-%d", today, name, suffix)
		} else {
			folderName = fmt.Sprintf("%s-%d", name, suffix)
		}
		targetPath = filepath.Join(baseDir, folderName)
		suffix++

		if suffix > 100 {
			return nil, fmt.Errorf("too many experiments with name %q", name)
		}
	}

	// Create the directory (including parents)
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create experiment directory: %w", err)
	}

	// Extract display name from folder name to be consistent with ListExperiments
	// When datePrefix is true, folderName is "YYYY-MM-DD-name", so strip the date prefix
	displayName := name
	if datePrefix && len(folderName) >= 11 && folderName[4] == '-' && folderName[7] == '-' && folderName[10] == '-' {
		displayName = folderName[11:]
	}

	exp := &Experiment{
		Name:    displayName,
		Path:    targetPath,
		Date:    time.Now(),
		HasDate: datePrefix,
		ModTime: time.Now(),
	}

	return exp, nil
}

// FindOrCreate finds an existing experiment by fuzzy match or creates a new one
// Returns (experiment, created, error)
func FindOrCreate(baseDir, query string, datePrefix bool) (*Experiment, bool, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, false, fmt.Errorf("failed to create experiments directory: %w", err)
	}

	// List existing experiments
	experiments, err := ListExperiments(baseDir)
	if err != nil {
		return nil, false, err
	}

	// Check for exact match first
	if exp := FindExact(experiments, query); exp != nil {
		return exp, false, nil
	}

	// Fuzzy search
	matches := FuzzyFind(experiments, query)

	// If there's a strong single match (only 1 result), use it
	// Find the matching experiment in the original slice to return a stable pointer
	if len(matches) == 1 {
		for i := range experiments {
			if experiments[i].Path == matches[0].Path {
				return &experiments[i], false, nil // Safe: points to slice element
			}
		}
	}

	// No good match - create new experiment
	exp, err := CreateExperiment(baseDir, query, datePrefix)
	if err != nil {
		return nil, false, err
	}

	return exp, true, nil
}
