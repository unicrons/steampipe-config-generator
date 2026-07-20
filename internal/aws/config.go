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

	creds, err := assumeRole(ctx, sts.NewFromConfig(awscfg), cfg.AssumeRoleArn, "steampipeConfigGenerator")
	if err != nil {
		return aws.Config{}, fmt.Errorf("assuming role %s: %w", cfg.AssumeRoleArn, err)
	}

	awscfg, err = awsconfig.LoadDefaultConfig(ctx,
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

func assumeRole(ctx context.Context, client *sts.Client, roleArn, sessionName string) (aws.Credentials, error) {
	provider := stscreds.NewAssumeRoleProvider(client, roleArn, func(o *stscreds.AssumeRoleOptions) {
		o.RoleSessionName = sessionName
	})

	creds, err := provider.Retrieve(ctx)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("retrieving assumed role credentials: %w", err)
	}
	return creds, nil
}
