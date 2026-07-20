package aws

// Account is a single AWS Organizations account as fetched from the AWS API, with its tags
// and parent organizational unit already resolved.
type Account struct {
	ID   string
	Name string
	OU   string
	Tags map[string]string
}
