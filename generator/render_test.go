package generator

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderCredentials(t *testing.T) {
	accounts := []Account{
		{Name: "team_foo", RoleARN: "arn:aws:iam::111111111111:role/my-role", CredentialSource: "Environment"},
	}

	var buf bytes.Buffer
	if err := RenderCredentials(&buf, accounts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"[team_foo]", "role_arn = arn:aws:iam::111111111111:role/my-role", "credential_source = Environment"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q, got:\n%s", want, out)
		}
	}
}

// TestRenderCredentials_NoHTMLEscaping guards against a regression to html/template: AWS
// account names can contain characters like "&" that html/template would escape, corrupting
// the generated .ini-style credentials file.
func TestRenderCredentials_NoHTMLEscaping(t *testing.T) {
	accounts := []Account{
		{Name: "r&d_team", RoleARN: "arn:aws:iam::111111111111:role/my-role", CredentialSource: "Environment"},
	}

	var buf bytes.Buffer
	if err := RenderCredentials(&buf, accounts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(buf.String(), "&amp;") {
		t.Fatalf("output was HTML-escaped, want raw text:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "[r&d_team]") {
		t.Fatalf("output missing unescaped account name, got:\n%s", buf.String())
	}
}

func TestRenderConnections_DefaultTemplate(t *testing.T) {
	accounts := []Account{
		{
			Name:          "team_foo",
			DefaultRegion: "us-east-1",
			ImportSchema:  "enabled",
			TargetRegions: []string{"*"},
			Tags:          map[string][]string{"sandbox_account": {"true"}},
		},
		{
			Name:          "team_bar",
			DefaultRegion: "us-east-1",
			ImportSchema:  "enabled",
			TargetRegions: []string{"*"},
		},
	}

	tmpl, err := ParseConnectionsTemplate("")
	if err != nil {
		t.Fatalf("unexpected error parsing default template: %v", err)
	}

	var buf bytes.Buffer
	if err := RenderConnections(&buf, accounts, tmpl); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	for _, want := range []string{`connection "aws_team_foo"`, `connection "aws_team_bar"`, `profile        = "team_foo"`} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q, got:\n%s", want, out)
		}
	}
}

func TestParseConnectionsTemplate_InvalidPath(t *testing.T) {
	_, err := ParseConnectionsTemplate("/no/such/template.tmpl")
	if err == nil {
		t.Fatal("expected an error for a nonexistent template path")
	}
}

func TestAggregateTags(t *testing.T) {
	accounts := []Account{
		{Name: "team_foo", Tags: map[string][]string{"sandbox_account": {"true"}}},
		{Name: "team_bar", Tags: map[string][]string{"sandbox_account": {"true"}}},
		{Name: "team_baz", Tags: map[string][]string{"sandbox_account": {"false"}}},
	}

	got := aggregateTags(accounts)

	want := []string{"team_foo", "team_bar"}
	names := got["sandbox_account,true"]
	if len(names) != len(want) {
		t.Fatalf("aggregateTags()[sandbox_account,true] = %v, want %v", names, want)
	}
	seen := map[string]bool{names[0]: true, names[1]: true}
	for _, w := range want {
		if !seen[w] {
			t.Errorf("aggregateTags()[sandbox_account,true] missing %q, got %v", w, names)
		}
	}

	if names := got["sandbox_account,false"]; len(names) != 1 || names[0] != "team_baz" {
		t.Errorf("aggregateTags()[sandbox_account,false] = %v, want [team_baz]", names)
	}
}
