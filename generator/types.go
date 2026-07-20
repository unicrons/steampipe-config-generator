package generator

// Account is an AWS Organizations account together with the data needed to render its
// Steampipe connection and credentials entries. Tags maps each tag key to its value(s) - a
// single-element slice for tags with no configured split, or multiple elements for tags
// listed in Options.TagSplit.
type Account struct {
	Name             string
	RoleARN          string
	CredentialSource string
	ImportSchema     string
	DefaultRegion    string
	TargetRegions    []string
	Tags             map[string][]string
}

// Options configures a Generator.
type Options struct {
	// AssumeRoleArn is the IAM role to assume before calling AWS Organizations, if any.
	AssumeRoleArn string
	// Region is the AWS region used both to call AWS Organizations and as each account's
	// DefaultRegion.
	Region string
	// RoleName is the IAM role name used to build each account's RoleARN.
	RoleName string
	// CredentialSource is the AWS credential source written for each account.
	CredentialSource string
	// ImportSchema controls the import_schema value written for each account.
	ImportSchema string
	// TargetRegions is the list of regions written for each account (["*"] for all).
	TargetRegions []string
	// SkipOUs lists organizational unit IDs whose accounts are excluded from the result.
	SkipOUs []string
	// TagSplit maps a tag key to the set of delimiter characters (e.g. ":-") its value
	// should be split on. Tags whose key isn't listed here keep their raw value, unchanged.
	// Only characters from AWS's supported tag character set are valid delimiters:
	// . : + = @ _ / -
	TagSplit map[string]string
}
