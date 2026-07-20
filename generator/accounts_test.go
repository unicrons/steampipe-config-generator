package generator

import (
	"context"
	"errors"
	"testing"

	internalaws "github.com/unicrons/steampipe-config-generator/internal/aws"
)

// fakeOrganizationsClient is an in-memory internalaws.OrganizationsClient - no AWS calls
// happen in these tests.
type fakeOrganizationsClient struct {
	accounts []internalaws.Account
	err      error
}

func (f *fakeOrganizationsClient) ListAccounts(ctx context.Context) ([]internalaws.Account, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.accounts, nil
}

func TestGenerator_Accounts(t *testing.T) {
	client := &fakeOrganizationsClient{
		accounts: []internalaws.Account{
			{ID: "111111111111", Name: "Team Foo", OU: "ou-root", Tags: map[string]string{"team": "foo"}},
			{ID: "222222222222", Name: "team-bar", OU: "ou-sandbox", Tags: map[string]string{"team": "bar"}},
		},
	}
	g := &generator{
		client: client,
		opts: Options{
			RoleName:         "my-role",
			CredentialSource: "Environment",
			ImportSchema:     "enabled",
			Region:           "us-east-1",
			TargetRegions:    []string{"*"},
			SkipOUs:          []string{"ou-sandbox"},
		},
	}

	accounts, err := g.Accounts(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(accounts) != 1 {
		t.Fatalf("got %d accounts, want 1 (ou-sandbox should be skipped): %+v", len(accounts), accounts)
	}

	got := accounts[0]
	if got.Name != "team_foo" {
		t.Errorf("Name = %q, want %q (normalized)", got.Name, "team_foo")
	}
	if got.RoleARN != "arn:aws:iam::111111111111:role/my-role" {
		t.Errorf("RoleARN = %q", got.RoleARN)
	}
	if want := []string{"foo"}; len(got.Tags["team"]) != 1 || got.Tags["team"][0] != want[0] {
		t.Errorf("Tags[team] = %v, want %v", got.Tags["team"], want)
	}
}

func TestGenerator_Accounts_FetchErrorIsNotSilenced(t *testing.T) {
	wantErr := errors.New("TooManyRequestsException")
	client := &fakeOrganizationsClient{err: wantErr}
	g := &generator{client: client, opts: Options{}}

	_, err := g.Accounts(t.Context())
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want it to wrap %v", err, wantErr)
	}
}
