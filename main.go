package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/unicrons/steampipe-config-generator/cmd"
	"github.com/unicrons/steampipe-config-generator/generator"
)

// newGenerator abstracts generator.New so tests can inject a fake Generator instead of
// hitting real AWS. Production code always passes generator.New itself.
type newGeneratorFunc func(ctx context.Context, opts generator.Options) (generator.Generator, error)

func run(ctx context.Context, log *slog.Logger, flags *cmd.Flags, newGenerator newGeneratorFunc) error {
	gen, err := newGenerator(ctx, generator.Options{
		AssumeRoleArn:    flags.AssumeRoleArn,
		Region:           flags.DefaultRegion,
		RoleName:         flags.RoleName,
		CredentialSource: flags.CredentialSource,
		ImportSchema:     flags.ImportSchema,
		TargetRegions:    flags.TargetRegions,
		SkipOUs:          flags.SkipOUs,
		TagSplit:         flags.TagSplit,
	})
	if err != nil {
		return fmt.Errorf("creating generator: %w", err)
	}

	accounts, err := gen.Accounts(ctx)
	if err != nil {
		return err
	}

	credentialsFile := filepath.Join(flags.CredentialPath, "credentials")
	if err := writeCredentialsFile(flags.CredentialPath, accounts); err != nil {
		return err
	}
	log.Info("wrote AWS credentials file", "path", credentialsFile)

	connectionsFile := filepath.Join(flags.ConnectionsPath, "aws.spc")
	if err := writeConnectionsFile(flags.ConnectionsPath, flags.TemplatePath, accounts); err != nil {
		return err
	}
	log.Info("wrote Steampipe connections file", "path", connectionsFile)

	log.Info("config files created successfully")
	return nil
}

func writeCredentialsFile(path string, accounts []generator.Account) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return fmt.Errorf("creating aws credentials path: %w", err)
	}

	file, err := os.Create(filepath.Join(path, "credentials"))
	if err != nil {
		return fmt.Errorf("creating aws credentials file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if err := generator.RenderCredentials(file, accounts); err != nil {
		return fmt.Errorf("rendering aws credentials file: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("closing aws credentials file: %w", err)
	}
	return nil
}

func writeConnectionsFile(path, templatePath string, accounts []generator.Account) error {
	tmpl, err := generator.ParseConnectionsTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("parsing connections template: %w", err)
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return fmt.Errorf("creating aws connections path: %w", err)
	}

	file, err := os.Create(filepath.Join(path, "aws.spc"))
	if err != nil {
		return fmt.Errorf("creating aws connections file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if err := generator.RenderConnections(file, accounts, tmpl); err != nil {
		return fmt.Errorf("rendering aws connections file: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("closing aws connections file: %w", err)
	}
	return nil
}

func main() {
	root := cmd.NewRootCmd(func(ctx context.Context, log *slog.Logger, flags *cmd.Flags) error {
		return run(ctx, log, flags, generator.New)
	})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
