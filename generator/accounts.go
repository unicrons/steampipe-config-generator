package generator

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// validTagSplitDelimiters is the subset of AWS's supported tag character set that may be used
// as a --tagSplit delimiter: letters, numbers, and ". : + = @ _ / -".
const validTagSplitDelimiters = ".:+=@_/-"

// parseDelimiters parses a --tagSplit value for one key (e.g. ":,-") into the set of
// individual delimiter characters to split on. Delimiters are comma-separated; cmd parses each
// --tagSplit occurrence on the first "=" only (see cmd.parseTagSplit), so a delimiter list here
// may itself contain "=" or "," without ambiguity.
func parseDelimiters(key, raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	var delimiters strings.Builder
	for _, token := range strings.Split(raw, ",") {
		runes := []rune(token)
		if len(runes) != 1 {
			return "", fmt.Errorf("tag %q: delimiter %q must be a single character", key, token)
		}
		if !strings.ContainsRune(validTagSplitDelimiters, runes[0]) {
			return "", fmt.Errorf("tag %q: delimiter %q is not a valid AWS tag character; valid delimiters are %q", key, token, validTagSplitDelimiters)
		}
		delimiters.WriteRune(runes[0])
	}
	return delimiters.String(), nil
}

// validateTagSplit rejects any configured delimiter character outside validTagSplitDelimiters.
// It also rejects an empty tag key: AWS tag keys can never be empty, so one showing up here is
// never something the caller actually meant (e.g. --tagSplit="=:,-", a key accidentally left
// blank).
func validateTagSplit(tagSplit map[string]string) error {
	for key, raw := range tagSplit {
		if key == "" {
			return fmt.Errorf("--tagSplit has an entry with an empty tag key")
		}
		if _, err := parseDelimiters(key, raw); err != nil {
			return err
		}
	}
	return nil
}

// splitTagValue returns the individual values for a tag. If key has no configured delimiters
// in tagSplit, value is returned unchanged (legacy single-value behavior). Otherwise value is
// split on any of its configured delimiter characters, with whitespace trimmed and empty
// sub-values dropped. tagSplit is assumed already validated (see validateTagSplit, called from
// New before any AWS call), so a parse error here is treated defensively as a no-op.
func splitTagValue(key, value string, tagSplit map[string]string) []string {
	raw, ok := tagSplit[key]
	if !ok {
		return []string{value}
	}

	delimiters, err := parseDelimiters(key, raw)
	if err != nil {
		return []string{value}
	}

	parts := strings.FieldsFunc(value, func(r rune) bool {
		return strings.ContainsRune(delimiters, r)
	})

	values := make([]string, 0, len(parts))
	for _, p := range parts {
		if p := strings.TrimSpace(p); p != "" {
			values = append(values, p)
		}
	}
	return values
}

func (g *generator) Accounts(ctx context.Context) ([]Account, error) {
	orgAccounts, err := g.client.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching organization accounts: %w", err)
	}

	accounts := make([]Account, 0, len(orgAccounts))
	for _, acc := range orgAccounts {
		if slices.Contains(g.opts.SkipOUs, acc.OU) {
			continue
		}

		tags := make(map[string][]string, len(acc.Tags))
		for key, value := range acc.Tags {
			tags[key] = splitTagValue(key, value, g.opts.TagSplit)
		}

		accounts = append(accounts, Account{
			Name:             normalizeAccountName(acc.Name),
			RoleARN:          fmt.Sprintf("arn:aws:iam::%s:role/%s", acc.ID, g.opts.RoleName),
			CredentialSource: g.opts.CredentialSource,
			ImportSchema:     g.opts.ImportSchema,
			DefaultRegion:    g.opts.Region,
			TargetRegions:    g.opts.TargetRegions,
			Tags:             tags,
		})
	}

	return accounts, nil
}

func normalizeAccountName(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, " ", "_"), "-", "_"))
}
