package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/unicrons/steampipe-config-generator/internal/logger"
)

// Flags holds the parsed and validated values of the root command's flags, ready to be
// consumed by the injected run function.
type Flags struct {
	RoleName         string
	CredentialSource string
	CredentialPath   string
	ConnectionsPath  string
	ImportSchema     string
	DefaultRegion    string
	TargetRegions    []string
	AssumeRoleArn    string
	TemplatePath     string
	LogFormat        string
	SkipOUs          []string
	TagSplit         map[string]string
}

var (
	validCredentialSources = []string{"Ec2InstanceMetadata", "Environment", "EcsContainer"}
	validImportSchemas     = []string{"enabled", "disabled"}
	validLogFormats        = []string{"default", "json"}
)

// NewRootCmd builds the root command. run is invoked with the request context, a logger
// configured for the requested --log format, and the fully validated flags.
func NewRootCmd(run func(ctx context.Context, log *slog.Logger, flags *Flags) error) *cobra.Command {
	var (
		flags         Flags
		targetRegions string
		skipOUs       string
		rawTagSplit   []string
	)

	cmd := &cobra.Command{
		Use:          "steampipe-config-generator",
		Short:        "Generate Steampipe AWS connection config files from an AWS Organization",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFlagValues(&flags); err != nil {
				return err
			}

			tagSplit, err := parseTagSplit(rawTagSplit)
			if err != nil {
				return err
			}
			flags.TagSplit = tagSplit

			log := logger.New(flags.LogFormat)

			if err := applyFlagDefaults(log, &flags, targetRegions, skipOUs); err != nil {
				return err
			}

			return run(cmd.Context(), log, &flags)
		},
	}

	cmd.Flags().StringVar(&flags.RoleName, "role", "", "AWS Role to use in AWS config credentials")
	cmd.Flags().StringVar(&flags.CredentialSource, "credential", "Environment", "AWS Credential source. Valid values are: Ec2InstanceMetadata, Environment, EcsContainer")
	cmd.Flags().StringVar(&flags.CredentialPath, "path", "", "AWS Credentials file path")
	cmd.Flags().StringVar(&flags.ConnectionsPath, "connections", "", "Steampipe AWS connections file path")
	cmd.Flags().StringVar(&flags.ImportSchema, "schema", "enabled", "AWS Connection import schema. Valid values are: enabled, disabled")
	cmd.Flags().StringVar(&flags.DefaultRegion, "region", "", "AWS Connection default region")
	cmd.Flags().StringVar(&targetRegions, "regions", "all", "AWS Connection target regions")
	cmd.Flags().StringVar(&flags.AssumeRoleArn, "assume", "", "AWS Role to assume for getting Organization accounts")
	cmd.Flags().StringVar(&flags.TemplatePath, "template", "", "Custom connections template path")
	cmd.Flags().StringVar(&flags.LogFormat, "log", "default", "Log format: default, json")
	cmd.Flags().StringVar(&skipOUs, "skipOUs", "", "AWS OU IDs to skip from account connections")
	cmd.Flags().StringArrayVar(&rawTagSplit, "tagSplit", nil, `Per-tag delimiter character(s) to split a multi-value tag on, as key=delimiter[,delimiter...] (repeatable), e.g. --tagSplit="team=:,-" splits the "team" tag on ':' or '-'. Parsed on the first '=' only, so delimiters may include '=' itself.`)

	if err := cmd.MarkFlagRequired("role"); err != nil {
		panic(err)
	}

	cmd.Version = fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date)
	cmd.SetVersionTemplate("steampipe-config-generator {{.Version}}\n")

	cmd.AddCommand(NewVersionCmd())

	return cmd
}

// parseTagSplit parses each --tagSplit occurrence (one per tag key) into a map. It's parsed by
// hand, splitting on only the first "=", rather than via pflag.StringToStringVar: that type
// counts every "=" in a single occurrence to decide whether to parse it as one key=value pair
// or several comma-separated ones, so a delimiter list containing "=" itself (a valid AWS tag
// character) would be misparsed into a bogus extra entry instead of failing loudly. Splitting on
// the first "=" only sidesteps that entirely, since --tagSplit is already one key per occurrence.
func parseTagSplit(raw []string) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	tagSplit := make(map[string]string, len(raw))
	for _, entry := range raw {
		key, delimiters, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("--tagSplit %q must be formatted as key=delimiter[,delimiter...]", entry)
		}
		tagSplit[key] = delimiters
	}
	return tagSplit, nil
}

func validateFlagValues(flags *Flags) error {
	if !slices.Contains(validCredentialSources, flags.CredentialSource) {
		return fmt.Errorf("--credential flag doesn't contain a valid value")
	}
	if !slices.Contains(validImportSchemas, flags.ImportSchema) {
		return fmt.Errorf("--schema flag doesn't contain a valid value")
	}
	if !slices.Contains(validLogFormats, flags.LogFormat) {
		return fmt.Errorf("--log unknown value. Valid values are: default, json")
	}
	return nil
}

// applyFlagDefaults fills in the defaults and derived fields that depend on the environment
// (home directory, AWS_REGION) or on other flags (regions, skipOUs).
func applyFlagDefaults(log *slog.Logger, flags *Flags, targetRegions, skipOUs string) error {
	if flags.CredentialPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting user's home directory: %w", err)
		}
		flags.CredentialPath = filepath.Join(homeDir, ".aws/")
	}

	if flags.ConnectionsPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting user's home directory: %w", err)
		}
		flags.ConnectionsPath = filepath.Join(homeDir, ".steampipe/config/")
	}

	if flags.DefaultRegion == "" {
		flags.DefaultRegion = os.Getenv("AWS_REGION")
		if flags.DefaultRegion == "" {
			flags.DefaultRegion = "us-east-1"
			log.Info("default region not defined, using default", "region", flags.DefaultRegion)
		} else {
			log.Debug("default region not defined, using value from env AWS_REGION", "region", flags.DefaultRegion)
		}
	}

	if targetRegions == "all" {
		flags.TargetRegions = []string{"*"}
	} else {
		flags.TargetRegions = strings.Split(targetRegions, ",")
	}
	log.Debug("regions", "value", flags.TargetRegions)

	flags.SkipOUs = strings.Split(skipOUs, ",")
	log.Debug("skipOUs", "value", flags.SkipOUs)

	return nil
}
