# CloudCutter

CloudCutter is a powerful, Go-based command-line tool designed to normalise, search, and analyse Microsoft Purview CSV exports. It provides advanced querying capabilities, Sigma rule integration, and chronological data handling to streamline digital forensics and incident response (DFIR) workflows.

## Features

- **Advanced Search Engine**: Query your audit logs using a flexible expression language.
  - **Nested Field Support**: Access deep data structures with dot notation (e.g., `Emails.Subject`).
  - **SQL-style Wildcards**: Use `*` and `%` for pattern matching within the `LIKE` operator.
  - **Array Handling**: Automatically applies "any match" logic when querying lists of items.
- **Chronological Comparisons**: Intelligently parses and compares `Date` and `Time` fields as chronological values rather than simple strings.
- **Sigma Rule Integration**: Apply standard Sigma rules to your Purview data to detect known threat patterns.
- **Data Normalisation**: Automatically promotes nested JSON fields (like `AuditData`) to top-level attributes for easier querying and display.
- **Global Debug Logging**: Detailed execution tracing with options to log to `stderr` or a dedicated file.
- **Customisable Formatting**: View results in a clean, human-readable log format or as raw JSON.

## Installation

Ensure you have [Go](https://go.dev/doc/install) installed, then clone the repository and build the binary:

```powershell
go build -o CloudCutter.exe main.go
```

## Usage

CloudCutter offers two primary commands: `search` and `analyse`.

### Searching Logs

Use the `search` command to filter events based on specific criteria.

```powershell
.\CloudCutter.exe search -f "audit_export.csv" -q "UserID == 'admin@example.com' AND Date > '2024-01-01'"
```

#### Search Operator Examples:
- **Nested Fields**: `Emails.Subject LIKE "*Invoice*"`
- **Wildcards**: `ClientIP LIKE "192.168.*"`
- **Chronological**: `Time >= "13:00:00" AND Time <= "14:00:00"`
- **Existence**: `Files.FileName != ""`

### Analysing with Sigma Rules

Use the `analyse` command to scan your logs against a directory of Sigma rules.

```powershell
.\CloudCutter.exe analyse -f "audit_export.csv" -s "./rules/m365"
```

### Global Flags

- `-f, --file`: Path to the Microsoft Purview CSV export (required).
- `-d, --debug`: Enable verbose debug logging to `stderr`.
- `--log-file`: Path to a file where debug logs should be saved.
- `--limit`: Limit the number of results displayed.

## Troubleshooting

### PowerShell Quoting
PowerShell can sometimes strip double quotes from your search query. If you encounter issues with queries containing spaces, use single quotes inside double quotes for the query string:

**Correct:**
`.\CloudCutter.exe search ... -q "Emails.Subject == 'Urgent Request'"`

**Incorrect:**
`.\CloudCutter.exe search ... -q 'Emails.Subject == "Urgent Request"'` (PowerShell may strip the inner quotes)

## Output Formats
Results can be displayed in several formats using the `--format` flag:
- `log`: A clean, vertical representation of key event fields (default).
- `json`: The raw JSON representation of the event.

## License
[Insert License Information Here]
