package aws

import "context"

// Account is a single AWS Organizations account as fetched from the AWS API, with its tags
// and parent organizational unit already resolved.
type Account struct {
	ID   string
	Name string
	OU   string
	Tags map[string]string
}

// OrganizationsClient lists AWS Organizations accounts. It exposes only the operation package
// generator needs, rather than mirroring the AWS SDK client, so tests can fake it without
// calling AWS.
type OrganizationsClient interface {
	// ListAccounts returns all ACTIVE accounts in the organization, with tags and OU populated.
	ListAccounts(ctx context.Context) ([]Account, error)
}
