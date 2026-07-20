package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

// fakeOrganizationsAPI is an in-memory organizationsAPI - no AWS calls happen in these tests.
// Each method returns a single page (our real pagination loops work the same either way, since
// they just stop as soon as NextToken is nil).
type fakeOrganizationsAPI struct {
	accounts    []types.Account
	accountsErr error

	tags    map[string][]types.Tag
	tagsErr map[string]error

	parents    map[string][]types.Parent
	parentsErr map[string]error
}

func (f *fakeOrganizationsAPI) ListAccounts(ctx context.Context, params *organizations.ListAccountsInput, optFns ...func(*organizations.Options)) (*organizations.ListAccountsOutput, error) {
	if f.accountsErr != nil {
		return nil, f.accountsErr
	}
	return &organizations.ListAccountsOutput{Accounts: f.accounts}, nil
}

func (f *fakeOrganizationsAPI) ListTagsForResource(ctx context.Context, params *organizations.ListTagsForResourceInput, optFns ...func(*organizations.Options)) (*organizations.ListTagsForResourceOutput, error) {
	id := *params.ResourceId
	if err := f.tagsErr[id]; err != nil {
		return nil, err
	}
	return &organizations.ListTagsForResourceOutput{Tags: f.tags[id]}, nil
}

func (f *fakeOrganizationsAPI) ListParents(ctx context.Context, params *organizations.ListParentsInput, optFns ...func(*organizations.Options)) (*organizations.ListParentsOutput, error) {
	id := *params.ChildId
	if err := f.parentsErr[id]; err != nil {
		return nil, err
	}
	return &organizations.ListParentsOutput{Parents: f.parents[id]}, nil
}

func strPtr(s string) *string { return &s }

func TestOrganizationsClient_ListAccounts(t *testing.T) {
	api := &fakeOrganizationsAPI{
		accounts: []types.Account{
			{Id: strPtr("111111111111"), Name: strPtr("Team Foo"), State: types.AccountStateActive},
			{Id: strPtr("222222222222"), Name: strPtr("Team Bar"), State: types.AccountStateSuspended},
			{Id: strPtr("333333333333"), Name: strPtr("Team Baz"), State: types.AccountStateActive},
		},
		tags: map[string][]types.Tag{
			"111111111111": {{Key: strPtr("team"), Value: strPtr("foo")}},
			"333333333333": {{Key: strPtr("team"), Value: strPtr("baz")}},
		},
		parents: map[string][]types.Parent{
			"111111111111": {{Id: strPtr("ou-root")}},
			"333333333333": {{Id: strPtr("ou-sandbox")}},
		},
	}
	c := &organizationsClient{client: api}

	accounts, err := c.ListAccounts(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(accounts) != 2 {
		t.Fatalf("got %d accounts, want 2 (only ACTIVE): %+v", len(accounts), accounts)
	}

	byID := make(map[string]Account, len(accounts))
	for _, acc := range accounts {
		byID[acc.ID] = acc
	}

	foo, ok := byID["111111111111"]
	if !ok {
		t.Fatal("missing account 111111111111")
	}
	if foo.Name != "Team Foo" {
		t.Errorf("Name = %q, want %q", foo.Name, "Team Foo")
	}
	if foo.OU != "ou-root" {
		t.Errorf("OU = %q, want %q", foo.OU, "ou-root")
	}
	if foo.Tags["team"] != "foo" {
		t.Errorf(`Tags["team"] = %q, want %q`, foo.Tags["team"], "foo")
	}

	if _, ok := byID["222222222222"]; ok {
		t.Error("suspended account 222222222222 should have been excluded")
	}
}

func TestOrganizationsClient_ListAccounts_ListAccountsError(t *testing.T) {
	wantErr := errors.New("TooManyRequestsException")
	c := &organizationsClient{client: &fakeOrganizationsAPI{accountsErr: wantErr}}

	_, err := c.ListAccounts(t.Context())
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestOrganizationsClient_ListAccounts_TagsErrorIsNotSilenced(t *testing.T) {
	wantErr := errors.New("AccessDenied")
	api := &fakeOrganizationsAPI{
		accounts: []types.Account{
			{Id: strPtr("111111111111"), Name: strPtr("Team Foo"), State: types.AccountStateActive},
		},
		tagsErr:    map[string]error{"111111111111": wantErr},
		parents:    map[string][]types.Parent{"111111111111": {{Id: strPtr("ou-root")}}},
		parentsErr: map[string]error{},
	}
	c := &organizationsClient{client: api}

	_, err := c.ListAccounts(t.Context())
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestOrganizationsClient_ListAccounts_OUErrorIsNotSilenced(t *testing.T) {
	wantErr := errors.New("AccessDenied")
	api := &fakeOrganizationsAPI{
		accounts: []types.Account{
			{Id: strPtr("111111111111"), Name: strPtr("Team Foo"), State: types.AccountStateActive},
		},
		tags:       map[string][]types.Tag{"111111111111": nil},
		parentsErr: map[string]error{"111111111111": wantErr},
	}
	c := &organizationsClient{client: api}

	_, err := c.ListAccounts(t.Context())
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestOrganizationsClient_GetAccountOU_NoParent(t *testing.T) {
	api := &fakeOrganizationsAPI{parents: map[string][]types.Parent{}}
	c := &organizationsClient{client: api}

	_, err := c.getAccountOU(t.Context(), "111111111111")
	if err == nil {
		t.Fatal("expected an error when an account has no parent OU")
	}
}
