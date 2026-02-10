package format

import (
	// Standard library dependencies
	"fmt"

	// Internal dependencies
	"CloudCutter/models"
)

func FormatEvent(event models.PurviewEvent, format string) string {
	switch format {
	case "log":
		return logFormat(event)
	default:
		return logFormat(event)
	}
}

func logFormat(event models.PurviewEvent) string {
	return fmt.Sprintf("%s %s %s %s", event.Timestamp, event.Operation, event.UserID, event.ClientIP)
}
