package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Config configures LoadConfig.
type Config struct {
	// AssumeRoleArn, if set, is assumed before any Organizations call.
	AssumeRoleArn string
	// Region is the AWS region used when AssumeRoleArn is set.
	Region string
}

// LoadConfig returns an AWS SDK config for cfg, assuming cfg.AssumeRoleArn first if set.
func LoadConfig(ctx context.Context, cfg Config) (aws.Config, error) {
	awscfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return aws.Config{}, fmt.Errorf("loading aws config: %w", err)
	}

	if cfg.AssumeRoleArn == "" {
		return awscfg, nil
	}

	return withAssumedRole(ctx, sts.NewFromConfig(awscfg), cfg)
}

// withAssumedRole assumes cfg.AssumeRoleArn via client and returns a config using the
// resulting temporary credentials. Split out from LoadConfig so it's testable with a fake STS
// client - LoadConfig itself is left with nothing but the (untestable without a real network
// call) construction of the real *sts.Client.
func withAssumedRole(ctx context.Context, client stscreds.AssumeRoleAPIClient, cfg Config) (aws.Config, error) {
	creds, err := assumeRole(ctx, client, cfg.AssumeRoleArn, "steampipeConfigGenerator")
	if err != nil {
		return aws.Config{}, fmt.Errorf("assuming role %s: %w", cfg.AssumeRoleArn, err)
	}

	awscfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken,
		)),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("loading aws config for assumed role: %w", err)
	}

	return awscfg, nil
}

// assumeRole retrieves temporary credentials for roleArn. client only needs to implement the
// SDK's own stscreds.AssumeRoleAPIClient interface, not the concrete *sts.Client, so tests can
// fake the AssumeRole call without any network access.
func assumeRole(ctx context.Context, client stscreds.AssumeRoleAPIClient, roleArn, sessionName string) (aws.Credentials, error) {
	provider := stscreds.NewAssumeRoleProvider(client, roleArn, func(o *stscreds.AssumeRoleOptions) {
		o.RoleSessionName = sessionName
	})

	creds, err := provider.Retrieve(ctx)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("retrieving assumed role credentials: %w", err)
	}
	return creds, nil
}
