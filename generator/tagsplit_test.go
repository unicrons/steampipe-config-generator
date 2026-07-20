package generator

import (
	"strings"
	"testing"

	internalaws "github.com/unicrons/steampipe-config-generator/internal/aws"
)

// Case 1: a tag key with no configured delimiter keeps today's exact-match behavior.
func TestSplitTagValue_NoDelimiterConfigured(t *testing.T) {
	got := splitTagValue("team", "frontend:backend", nil)
	want := []string{"frontend:backend"}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("splitTagValue() = %v, want %v", got, want)
	}
}

// Case 2: a tag key with one configured delimiter splits into N map entries.
func TestSplitTagValue_OneDelimiter(t *testing.T) {
	got := splitTagValue("team", "frontend:backend", map[string]string{"team": ":"})
	want := []string{"frontend", "backend"}
	if !equalUnordered(got, want) {
		t.Errorf("splitTagValue() = %v, want %v", got, want)
	}
}

// Case 3: multiple configured delimiter characters (comma-separated in the flag value) split
// on any of them.
func TestSplitTagValue_MultipleDelimiters(t *testing.T) {
	got := splitTagValue("team", "frontend:backend-platform", map[string]string{"team": ":,-"})
	want := []string{"frontend", "backend", "platform"}
	if !equalUnordered(got, want) {
		t.Errorf("splitTagValue() = %v, want %v", got, want)
	}
}

// A comma-separated delimiter list parses correctly even though pflag.StringToStringVar also
// uses "," as its own pair separator - confirmed safe because a single --tagSplit occurrence
// with exactly one "=" never enters pflag's CSV parser (see TestNewRootCmd_TagSplit_CommaInValue).
func TestParseDelimiters_CommaSeparated(t *testing.T) {
	got, err := parseDelimiters("team", ":,-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalUnordered(strings.Split(got, ""), []string{":", "-"}) {
		t.Errorf("parseDelimiters() = %q, want the characters ':' and '-'", got)
	}
}

func TestParseDelimiters_MultiCharacterTokenRejected(t *testing.T) {
	_, err := parseDelimiters("team", "::")
	if err == nil {
		t.Fatal("expected an error for a multi-character delimiter token")
	}
}

// Case 4: an invalid delimiter character is rejected with a clear error, before any AWS call.
func TestValidateTagSplit_InvalidDelimiter(t *testing.T) {
	err := validateTagSplit(map[string]string{"team": "!"})
	if err == nil {
		t.Fatal("expected an error for an invalid delimiter character")
	}
	if !strings.Contains(err.Error(), "team") {
		t.Errorf("error should mention the offending tag key, got: %v", err)
	}
}

func TestNew_InvalidTagSplitDelimiter(t *testing.T) {
	_, err := New(t.Context(), Options{
		RoleName: "my-role",
		TagSplit: map[string]string{"team": "!"},
	})
	if err == nil {
		t.Fatal("expected an error for an invalid delimiter character")
	}
}

// Case 5: empty and whitespace-only sub-values (trailing/double delimiters) are dropped.
func TestSplitTagValue_DropsEmptyAndWhitespaceSubValues(t *testing.T) {
	got := splitTagValue("team", "frontend::  :backend:", map[string]string{"team": ":"})
	want := []string{"frontend", "backend"}
	if !equalUnordered(got, want) {
		t.Errorf("splitTagValue() = %v, want %v", got, want)
	}
}

// A key with a configured delimiter whose value doesn't actually contain that delimiter still
// goes through the split path, but yields a single unchanged value - same visible result as
// the no-delimiter-configured case, via a different code path.
func TestSplitTagValue_ConfiguredDelimiterNotPresentInValue(t *testing.T) {
	got := splitTagValue("team", "frontend", map[string]string{"team": ":"})
	want := []string{"frontend"}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("splitTagValue() = %v, want %v", got, want)
	}
}

// An empty delimiter set for a key is valid (zero characters to reject) and acts as a no-op:
// nothing matches, so the value is returned whole, same as if the key weren't configured.
func TestSplitTagValue_EmptyDelimiterSetIsNoOp(t *testing.T) {
	if err := validateTagSplit(map[string]string{"team": ""}); err != nil {
		t.Fatalf("unexpected error for an empty delimiter set: %v", err)
	}

	got := splitTagValue("team", "frontend:backend", map[string]string{"team": ""})
	want := []string{"frontend:backend"}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("splitTagValue() = %v, want %v", got, want)
	}
}

// Two different tag keys can have two different, independent delimiter sets configured at
// the same time, without bleeding into each other.
func TestSplitTagValue_MultipleKeysWithDifferentDelimiters(t *testing.T) {
	tagSplit := map[string]string{"team": ":", "cost_center": "+"}

	gotTeam := splitTagValue("team", "frontend:backend", tagSplit)
	if want := []string{"frontend", "backend"}; !equalUnordered(gotTeam, want) {
		t.Errorf(`splitTagValue("team", ...) = %v, want %v`, gotTeam, want)
	}

	gotCostCenter := splitTagValue("cost_center", "100+200", tagSplit)
	if want := []string{"100", "200"}; !equalUnordered(gotCostCenter, want) {
		t.Errorf(`splitTagValue("cost_center", ...) = %v, want %v`, gotCostCenter, want)
	}
}

// End-to-end: a multi-value tag resolves into two separate aggregator groups, replacing the
// single combined-value entry, while an unrelated tag keeps its legacy single-entry behavior.
func TestGenerator_Accounts_MultiValueTagAggregation(t *testing.T) {
	client := &fakeOrganizationsClient{
		accounts: []internalaws.Account{
			{ID: "111111111111", Name: "account-a", Tags: map[string]string{"team": "frontend:backend", "env": "prod"}},
			{ID: "222222222222", Name: "account-b", Tags: map[string]string{"team": "backend"}},
		},
	}
	g := &generator{
		client: client,
		opts:   Options{RoleName: "my-role", TagSplit: map[string]string{"team": ":"}},
	}

	accounts, err := g.Accounts(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tagged := aggregateTags(accounts)

	if _, ok := tagged["team,frontend:backend"]; ok {
		t.Error("the combined value entry should not exist - it must be replaced by the split entries")
	}
	if names := tagged["team,frontend"]; !equalUnordered(names, []string{"account_a"}) {
		t.Errorf(`tagged["team,frontend"] = %v, want [account_a]`, names)
	}
	if names := tagged["team,backend"]; !equalUnordered(names, []string{"account_a", "account_b"}) {
		t.Errorf(`tagged["team,backend"] = %v, want [account_a account_b]`, names)
	}
	if names := tagged["env,prod"]; !equalUnordered(names, []string{"account_a"}) {
		t.Errorf(`tagged["env,prod"] = %v, want [account_a] (unrelated tag, no split configured)`, names)
	}
}

func equalUnordered(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int)
	for _, v := range got {
		seen[v]++
	}
	for _, v := range want {
		if seen[v] == 0 {
			return false
		}
		seen[v]--
	}
	return true
}
