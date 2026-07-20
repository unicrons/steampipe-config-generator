// Package generator fetches AWS Organizations accounts and renders Steampipe AWS credentials
// and connection config files for them.
package generator

import (
	"context"
	"fmt"

	internalaws "github.com/unicrons/steampipe-config-generator/internal/aws"
)

// Generator fetches AWS Organizations accounts for Steampipe config generation.
type Generator interface {
	// Accounts fetches active accounts, excluding any organizational unit listed in
	// Options.SkipOUs, with each account's tags attached.
	Accounts(ctx context.Context) ([]Account, error)
}

type generator struct {
	client internalaws.OrganizationsClient
	opts   Options
}

// New returns a Generator configured from the default AWS environment, assuming
// opts.AssumeRoleArn first if set.
func New(ctx context.Context, opts Options) (Generator, error) {
	cfg, err := internalaws.LoadConfig(ctx, internalaws.Config{
		AssumeRoleArn: opts.AssumeRoleArn,
		Region:        opts.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("loading aws config: %w", err)
	}

	return &generator{
		client: internalaws.NewOrganizationsClient(cfg),
		opts:   opts,
	}, nil
}
