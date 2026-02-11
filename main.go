package main

import (
	// Standard library dependencies
	"fmt"
	"os"

	// Internal dependencies
	"CloudCutter/internal/format"
	"CloudCutter/internal/logger"
	"CloudCutter/internal/output"
	"CloudCutter/internal/parser"
	"CloudCutter/tools/analysis"
	"CloudCutter/tools/search"

	// External dependencies
	"github.com/spf13/cobra"
)

// Global variables for flags
var csvFile string
var debug bool
var logFile string
var outputFile string

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

	rootCommand.PersistentFlags().StringVarP(&csvFile, "file", "f", "", "Path to the CSV file to process")
	rootCommand.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	rootCommand.PersistentFlags().StringVarP(&logFile, "log-file", "", "", "Path to the log file to write debug logs to")
	rootCommand.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Output file to write the findings to (CSV)")
	rootCommand.MarkPersistentFlagRequired("file")

	rootCommand.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		logger.Enabled = debug
		logger.LogPath = logFile
		if debug {
			logger.Debugf("Debug logging enabled")
			if logFile != "" {
				logger.Debugf("Logging to file: %s", logFile)
			}
		}
	}

	// Add subcommands to the root command
	rootCommand.AddCommand(analysisCommand())
	rootCommand.AddCommand(searchCommand())

	// Execute the root command & catch any errors
	if err := rootCommand.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func searchCommand() *cobra.Command {
	// Variables
	var searchQuery string
	var listColumns bool
	var outputFormat string
	var limit int
	var countOnly bool

	// Define command
	var command = &cobra.Command{
		Use:   "search",
		Short: "Search for a specific term in the CSV file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeSearch(cmd, args, searchQuery, listColumns, outputFormat, limit, countOnly)
		},
	}

	// Define flags
	var queryHelpText = "Search query to filter events. \n" +
		"Operators: ==, !=, >, <, >=, <=, LIKE, AND, OR \n" +
		"Fields:    Operation, UserID, ClientIP, etc. \n" +
		"Examples:\n" +
		"	-q \"Operation == 'MailItemsAccessed'\" \n" +
		"	-q \"Subject LIKE 'Urgent' AND UserID == [EMAIL_ADDRESS]'\" \n" +
		"	-q \"ClientIP != '[IP_ADDRESS]' AND (Operation == 'FileModified' OR Operation == 'MailItemAccessed')\" "

	command.Flags().StringVarP(&searchQuery, "query", "q", "", queryHelpText)
	command.Flags().BoolVarP(&listColumns, "list", "", false, "List all available columns in the CSV")
	command.Flags().StringVarP(&outputFormat, "format", "", "log", "Format to output the events in")
	command.Flags().IntVarP(&limit, "limit", "l", 0, "Limit the number of events to output")
	command.Flags().BoolVarP(&countOnly, "count", "c", false, "Count the number of events")

	return command
}

func executeSearch(cmd *cobra.Command, args []string, searchQuery string, listColumns bool, outputFormat string, limit int, countOnly bool) error {
	// Parse the CSV file & return events
	events := parser.ParsePurviewCSV(csvFile)

	// List columns from CSV
	if listColumns {
		columns := parser.GetPurviewEventColumns(events, false)

		fmt.Println("Available columns in the CSV:")
		fmt.Println("-----------------------")

		for _, column := range columns {
			fmt.Println(" - ", column)
		}

		return nil
	}

	// Perform search
	if searchQuery != "" {
		// If there are positional args, append them to the search query
		// This handles cases where PowerShell strips quotes and splits the query
		if len(args) > 0 {
			for _, arg := range args {
				searchQuery += " " + arg
			}
		}
		filteredEvents := search.Query(events, searchQuery)

		if len(filteredEvents) > 0 {
			// Export to CSV if output file is specified
			if outputFile != "" {
				err := output.ExportToCSV(filteredEvents, outputFile, false)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error exporting to CSV: %v\n", err)
				} else {
					fmt.Printf("Successfully exported %d events to %s\n", len(filteredEvents), outputFile)
				}
			}

			printedCount := 0
			for _, event := range filteredEvents {
				if limit > 0 && printedCount >= limit {
					break
				}

				if countOnly == false && outputFile == "" {
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

	return nil
}

func analysisCommand() *cobra.Command {
	// Variables
	var sigmaFilePath string
	var outputFormat string

	// Define command
	var command = &cobra.Command{
		Use:   "analyse",
		Short: "Analyse a CSV file using Sigma rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeAnalysis(cmd, args, sigmaFilePath, outputFormat)
		},
	}

	// Define flags
	command.Flags().StringVarP(&sigmaFilePath, "sigma", "s", "", "Path to the Sigma files")
	command.Flags().StringVarP(&outputFormat, "format", "", "log", "Format to output the events in")
	command.MarkPersistentFlagRequired("sigma")

	return command
}

func executeAnalysis(cmd *cobra.Command, args []string, sigmaFilePath string, outputFormat string) error {
	// Parse the CSV file & return events
	events := parser.ParsePurviewCSV(csvFile)

	// Analyse the events using Sigma rules
	filteredEvents := analysis.AnalysePurviewCSV(events, sigmaFilePath)

	// Check if there are any filtered events
	if len(filteredEvents) > 0 {
		// Export to CSV if output file is specified
		if outputFile != "" {
			err := output.ExportToCSV(filteredEvents, outputFile, true)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error exporting to CSV: %v\n", err)
			} else {
				fmt.Printf("Successfully exported %d events to %s\n", len(filteredEvents), outputFile)
			}
		}

		// Still print to terminal if not just exporting
		printedCount := 0

		for _, event := range filteredEvents {
			if outputFile == "" {
				fmt.Println(format.FormatEvent(event, outputFormat))
			}
			printedCount++
		}
	} else {
		fmt.Println("No matches found...")
	}

	return nil
}
