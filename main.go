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

func run(ctx context.Context, log *slog.Logger, flags *cmd.Flags) error {
	gen, err := generator.New(ctx, generator.Options{
		AssumeRoleArn:    flags.AssumeRoleArn,
		Region:           flags.DefaultRegion,
		RoleName:         flags.RoleName,
		CredentialSource: flags.CredentialSource,
		ImportSchema:     flags.ImportSchema,
		TargetRegions:    flags.TargetRegions,
		SkipOUs:          flags.SkipOUs,
	})
	if err != nil {
		return fmt.Errorf("creating generator: %w", err)
	}

	accounts, err := gen.Accounts(ctx)
	if err != nil {
		return err
	}

	if err := writeCredentialsFile(flags.CredentialPath, accounts); err != nil {
		return err
	}

	if err := writeConnectionsFile(flags.ConnectionsPath, flags.TemplatePath, accounts); err != nil {
		return err
	}

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
	defer file.Close()

	if err := generator.RenderCredentials(file, accounts); err != nil {
		return fmt.Errorf("rendering aws credentials file: %w", err)
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
	defer file.Close()

	if err := generator.RenderConnections(file, accounts, tmpl); err != nil {
		return fmt.Errorf("rendering aws connections file: %w", err)
	}
	return nil
}

func main() {
	if err := cmd.NewRootCmd(run).Execute(); err != nil {
		os.Exit(1)
	}
}
