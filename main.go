package main

import (
	// Standard library dependencies
	"fmt"
	"os"

	// Internal dependencies
	"CloudCutter/internal/format"
	"CloudCutter/internal/parser"
	"CloudCutter/tools/analysis"
	"CloudCutter/tools/search"

	// External dependencies
	"github.com/spf13/cobra"
)

func main() {
	var rootCommand = &cobra.Command{
		Use:   "CloudCutter",
		Short: "A Purview Log Analysis Tool",
		Long:  `Purview Analyser is a tool inspired by Chainsaw to analyse Microsoft Purview CSV exports using Sigma rules.`,
	}

	rootCommand.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})

	var analyseCommand = &cobra.Command{
		Use:   "analyse",
		Short: "Analyse a CSV file using Sigma rules",
	}

	var searchCommand = &cobra.Command{
		Use:   "search",
		Short: "Search for a specific term in the CSV file",
	}

	// Define flags
	// - Globals
	var csvFile string

	rootCommand.PersistentFlags().StringVarP(&csvFile, "file", "f", "", "Path to the CSV file to process")
	rootCommand.MarkPersistentFlagRequired("file")

	// - Analyse
	var sigmaFilePath string

	analyseCommand.Flags().StringVarP(&sigmaFilePath, "sigma", "s", "", "Path to the Sigma files")
	analyseCommand.MarkPersistentFlagRequired("sigma")

	// - Search
	var searchQuery string
	var listColumns bool
	var outputFormat string
	var limit int
	var countOnly bool
	var outputFile string

	searchCommand.Flags().StringVarP(&searchQuery, "query", "q", "", "Search query to filter events")
	searchCommand.Flags().BoolVarP(&listColumns, "list", "", false, "List all available columns in the CSV")
	searchCommand.Flags().StringVarP(&outputFormat, "format", "", "log", "Format to output the events in")
	searchCommand.Flags().IntVarP(&limit, "limit", "l", 0, "Limit the number of events to output")
	searchCommand.Flags().BoolVarP(&countOnly, "count", "c", false, "Count the number of events")
	searchCommand.Flags().StringVarP(&outputFile, "output", "o", "", "Output file to write the results to")

	// Command executions
	analyseCommand.Run = func(cmd *cobra.Command, args []string) {
		// Parse the CSV file & return events
		events := parser.ParsePurviewCSV(csvFile)

		filteredEvents := analysis.AnalysePurviewCSV(events, sigmaFilePath)

		if len(filteredEvents) > 0 {
			printedCount := 0
			for _, event := range filteredEvents {
				fmt.Println(format.FormatEvent(event, outputFormat))

				printedCount++
			}
		} else {
			fmt.Println("No matches found...")
		}
	}

	searchCommand.Run = func(cmd *cobra.Command, args []string) {
		// Parse the CSV file & return events
		events := parser.ParsePurviewCSV(csvFile)

		// List columns from CSV
		if listColumns {
			columns := parser.GetPurviewEventColumns(events)

			fmt.Println("Available columns in the CSV:")

			for _, column := range columns {
				fmt.Println(column)
				fmt.Println("")
			}

			return
		}

		// Perform search
		if searchQuery != "" {
			filteredEvents := search.Query(events, searchQuery)

			if len(filteredEvents) > 0 {
				printedCount := 0
				for _, event := range filteredEvents {
					if limit > 0 && printedCount >= limit {
						break
					}

					if countOnly == false {
						fmt.Println(format.FormatEvent(event, outputFormat))
					}

					printedCount++
				}

				if countOnly {
					fmt.Println(printedCount)
				}
			} else {
				fmt.Println("No matches found...")
			}
		}
	}

	// Add subcommands to the root command
	rootCommand.AddCommand(analyseCommand)
	rootCommand.AddCommand(searchCommand)

	// Execute the root command & catch any errors
	if err := rootCommand.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
