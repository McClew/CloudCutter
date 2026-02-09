package search

import (
	// Standard library dependencies
	"fmt"
	"regexp"

	// Internal dependencies
	"CloudCutter/models"
)

func Query(events []models.PurviewEvent, query string) []models.PurviewEvent {
	//var columns = parser.GetPurviewEventColumns(events)

	expression := regexp.MustCompile(`(?i)\s+(AND|OR)\s+`)
	queries := expression.Split(query, -1)

	fmt.Println(queries)

	return nil
}
