package generator

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

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

		accounts = append(accounts, Account{
			Name:             normalizeAccountName(acc.Name),
			RoleARN:          fmt.Sprintf("arn:aws:iam::%s:role/%s", acc.ID, g.opts.RoleName),
			CredentialSource: g.opts.CredentialSource,
			ImportSchema:     g.opts.ImportSchema,
			DefaultRegion:    g.opts.Region,
			TargetRegions:    g.opts.TargetRegions,
			Tags:             acc.Tags,
		})
	}

	return accounts, nil
}

func normalizeAccountName(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, " ", "_"), "-", "_"))
}
