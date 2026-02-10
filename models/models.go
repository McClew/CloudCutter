package models

// Normalised Purview log
type PurviewEvent struct {
	RecordID            string                 `json:"record_id"`
	Date                string                 `json:"date"`
	Time                string                 `json:"time"`
	Timestamp           string                 `json:"timestamp"`
	UserID              string                 `json:"user_id"`
	Organisation        string                 `json:"organisation"`
	M365Service         string                 `json:"m365_service"`
	Operation           string                 `json:"operation"`
	OperationProperties string                 `json:"operation_properties"`
	ClientIP            string                 `json:"client_ip"`
	ClientAppName       string                 `json:"client_app_name"`
	UserAgent           string                 `json:"user_agent"`
	SourceFile          string                 `json:"source_file"`
	RawData             map[string]interface{} `json:"raw_data"`   // Everything from the CSV row
	AuditData           map[string]interface{} `json:"audit_data"` // Parsed from JSON in AuditData column
	Flattened           map[string]interface{} `json:"flattened"`  // Combined map for Sigma matching
}
