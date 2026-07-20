package generator

import (
	"embed"
	"fmt"
	"io"
	"text/template"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

const defaultConnectionsTemplate = "templates/aws_connections.tmpl"

// connectionsTemplateData is the data passed to the connections template: the accounts
// themselves, plus a view of their tags aggregated into per-tag-value connection groups.
type connectionsTemplateData struct {
	Accounts []Account
	Tags     map[string][]string
}

// ParseConnectionsTemplate returns the connections template to render with: the embedded
// default if path is empty, or the template at path otherwise.
func ParseConnectionsTemplate(path string) (*template.Template, error) {
	if path == "" {
		return template.ParseFS(templatesFS, defaultConnectionsTemplate)
	}
	return template.ParseFiles(path)
}

// RenderConnections renders the Steampipe AWS connections file for accounts using tmpl.
func RenderConnections(w io.Writer, accounts []Account, tmpl *template.Template) error {
	data := connectionsTemplateData{
		Accounts: accounts,
		Tags:     aggregateTags(accounts),
	}

	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("rendering connections template: %w", err)
	}
	return nil
}

// RenderCredentials renders the AWS credentials file for accounts.
func RenderCredentials(w io.Writer, accounts []Account) error {
	tmpl, err := template.ParseFS(templatesFS, "templates/aws_credentials.tmpl")
	if err != nil {
		return fmt.Errorf("parsing credentials template: %w", err)
	}

	if err := tmpl.Execute(w, accounts); err != nil {
		return fmt.Errorf("rendering credentials template: %w", err)
	}
	return nil
}

// aggregateTags groups account names by "tagKey,tagValue", mirroring the historical template
// lookup convention (index .Tags "key,value").
func aggregateTags(accounts []Account) map[string][]string {
	tagged := make(map[string][]string)
	for _, acc := range accounts {
		for key, value := range acc.Tags {
			tagKey := key + "," + value
			tagged[tagKey] = append(tagged[tagKey], acc.Name)
		}
	}
	return tagged
}
