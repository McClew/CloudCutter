package analysis

import (
	// Standard library dependencies
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	// Internal dependencies
	"CloudCutter/models"

	// External dependencies
	"github.com/bradleyjkemp/sigma-go"
	"github.com/bradleyjkemp/sigma-go/evaluator"
)

func AnalysePurviewCSV(events []models.PurviewEvent, sigmaFilePath string) []models.PurviewEvent {
	var filteredEvents []models.PurviewEvent
	yamlFilePaths := getYAMLFiles(sigmaFilePath)

	ctx := context.Background()

	for _, file := range yamlFilePaths {
		contents, err := os.ReadFile(file)
		if err != nil {
			return nil
		}

		var rule, _ = sigma.ParseRule(contents)
		eval := evaluator.ForRule(rule)

		for _, event := range events {
			result, _ := eval.Matches(ctx, event.Flattened)

			if result.Match {
				event.SigmaRuleTitle = rule.Title
				event.SigmaRuleDescription = rule.Description
				event.SigmaRuleSeverity = rule.Level
				event.SigmaRuleTags = rule.Tags

				events = append(events, event)
			}
		}
	}

	return filteredEvents
}

// Helpers
// Get all YAML files from a path
func getYAMLFiles(root string) []string {
	var files []string

	filepath.WalkDir(root, func(path string, directory fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories, but keep searching inside them
		if directory.IsDir() {
			return nil
		}

		// Check for .yaml or .yml extension (case-insensitive)
		extension := strings.ToLower(filepath.Ext(path))
		if extension == ".yaml" || extension == ".yml" {
			files = append(files, path)
		}

		return nil
	})

	return files
}
