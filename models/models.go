package models

// EmailItem represents email metadata in Purview logs
type EmailItem struct {
	ID                string `json:"id"`
	Subject           string `json:"subject"`
	InternetMessageID string `json:"internet_message_id"`
	SizeInBytes       int64  `json:"size_in_bytes"`
	CreationTime      string `json:"creation_time"`
}

// FileItem represents file metadata in Purview logs (e.g., SharePoint/OneDrive)
type FileItem struct {
	FileName      string `json:"file_name"`
	FileExtension string `json:"file_extension"`
	RelativeURL   string `json:"relative_url"`
	SiteURL       string `json:"site_url"`
	ObjectID      string `json:"object_id"`
}

// Normalised Purview log
type PurviewEvent struct {
	RecordID             string         `json:"record_id"`
	Date                 string         `json:"date"`
	Time                 string         `json:"time"`
	Timestamp            string         `json:"timestamp"`
	SigmaRuleTitle       string         `json:"sigma_rule_title"`
	SigmaRuleDescription string         `json:"sigma_rule_description"`
	SigmaRuleSeverity    string         `json:"sigma_rule_severity"`
	SigmaRuleTags        []string       `json:"sigma_rule_tags"`
	UserID               string         `json:"user_id"`
	Organisation         string         `json:"organisation"`
	EventSource          string         `json:"event_source"`
	M365Service          string         `json:"m365_service"`
	Operation            string         `json:"operation"`
	OperationProperties  string         `json:"operation_properties"`
	ClientIP             string         `json:"client_ip"`
	ClientAppName        string         `json:"client_app_name"`
	Client               string         `json:"client"`
	UserAgent            string         `json:"user_agent"`
	ActorInfo            string         `json:"actor_info"`
	AffectedItems        string         `json:"affected_items"`
	Folders              string         `json:"folders"`
	Folder               string         `json:"folder"`
	DestinationFolder    string         `json:"destination_folder"`
	SourceFile           string         `json:"source_file"`
	Emails               []EmailItem    `json:"emails"`
	Files                []FileItem     `json:"files"`
	RawData              map[string]any `json:"raw_data"`   // Everything from the CSV row
	AuditData            map[string]any `json:"audit_data"` // Parsed from JSON in AuditData column
	Flattened            map[string]any `json:"flattened"`  // Combined map for Sigma matching
}
