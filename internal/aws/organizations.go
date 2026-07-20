package aws

import (
	"context"
	"fmt"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"golang.org/x/sync/errgroup"
)

// maxConcurrentTagFetches and maxConcurrentOUFetches bound the number of concurrent
// per-account calls to stay under AWS Organizations' rate limits.
const (
	maxConcurrentTagFetches = 8 // under 10 TPS, burst 15 limit
	maxConcurrentOUFetches  = 3 // under 5 TPS, burst 8 limit
)

type organizationsClient struct {
	client *organizations.Client
}

// NewOrganizationsClient returns an OrganizationsClient backed by the real AWS SDK, using an
// aggressive retry policy since AWS Organizations has strict rate limits.
func NewOrganizationsClient(cfg awssdk.Config) OrganizationsClient {
	cfg.Retryer = func() awssdk.Retryer {
		return retry.NewStandard(func(o *retry.StandardOptions) {
			o.MaxAttempts = 5
			o.Backoff = retry.NewExponentialJitterBackoff(time.Second)
		})
	}

	return &organizationsClient{client: organizations.NewFromConfig(cfg)}
}

func (c *organizationsClient) ListAccounts(ctx context.Context) ([]Account, error) {
	accounts, err := c.listActiveAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return c.fetchTags(ctx, accounts) })
	group.Go(func() error { return c.fetchOUs(ctx, accounts) })

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return accounts, nil
}

func (c *organizationsClient) listActiveAccounts(ctx context.Context) ([]Account, error) {
	var accounts []Account

	paginator := organizations.NewListAccountsPaginator(c.client, &organizations.ListAccountsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing accounts page: %w", err)
		}

		for _, acc := range page.Accounts {
			if acc.State != types.AccountStateActive {
				continue
			}
			accounts = append(accounts, Account{ID: *acc.Id, Name: *acc.Name})
		}
	}

	return accounts, nil
}

// fetchTags fills in the Tags field of each account, bounded by maxConcurrentTagFetches. The
// first real error cancels the fetch and is returned - a failed fetch no longer leaves an
// account with an empty Tags map silently.
func (c *organizationsClient) fetchTags(ctx context.Context, accounts []Account) error {
	return fetchConcurrently(ctx, len(accounts), maxConcurrentTagFetches, func(ctx context.Context, i int) error {
		tags, err := c.listAccountTags(ctx, accounts[i].ID)
		if err != nil {
			return fmt.Errorf("account %s: fetching tags: %w", accounts[i].ID, err)
		}
		accounts[i].Tags = tags
		return nil
	})
}

// fetchOUs fills in the OU field of each account, bounded by maxConcurrentOUFetches. The first
// real error cancels the fetch and is returned.
func (c *organizationsClient) fetchOUs(ctx context.Context, accounts []Account) error {
	return fetchConcurrently(ctx, len(accounts), maxConcurrentOUFetches, func(ctx context.Context, i int) error {
		ou, err := c.getAccountOU(ctx, accounts[i].ID)
		if err != nil {
			return fmt.Errorf("account %s: fetching OU: %w", accounts[i].ID, err)
		}
		accounts[i].OU = ou
		return nil
	})
}

func (c *organizationsClient) listAccountTags(ctx context.Context, accountID string) (map[string]string, error) {
	tags := make(map[string]string)

	paginator := organizations.NewListTagsForResourcePaginator(c.client, &organizations.ListTagsForResourceInput{
		ResourceId: &accountID,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing tags: %w", err)
		}
		for _, tag := range page.Tags {
			tags[*tag.Key] = *tag.Value
		}
	}

	return tags, nil
}

func (c *organizationsClient) getAccountOU(ctx context.Context, accountID string) (string, error) {
	resp, err := c.client.ListParents(ctx, &organizations.ListParentsInput{ChildId: &accountID})
	if err != nil {
		return "", fmt.Errorf("listing parents: %w", err)
	}
	if len(resp.Parents) == 0 {
		return "", fmt.Errorf("no parent OU found for account %s", accountID)
	}
	return *resp.Parents[0].Id, nil
}
