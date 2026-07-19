package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

// Flags holds the parsed and validated values of the root command's flags, ready to be
// consumed by the generator logic.
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
}

var (
	validCredentialSources = []string{"Ec2InstanceMetadata", "Environment", "EcsContainer"}
	validImportSchemas     = []string{"enabled", "disabled"}
	validLogFormats        = []string{"default", "json"}
)

// NewRootCmd builds the root command. run is invoked with the fully validated flags once
// the command's own validation passes.
func NewRootCmd(run func(*Flags) error) *cobra.Command {
	var (
		flags         Flags
		targetRegions string
		skipOUs       string
	)

	cmd := &cobra.Command{
		Use:           "steampipe-config-generator",
		Short:         "Generate Steampipe AWS connection config files from an AWS Organization",
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := finalizeFlags(&flags, targetRegions, skipOUs); err != nil {
				return err
			}

			log.Debug("parsed flags:", flags)

			return run(&flags)
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

	if err := cmd.MarkFlagRequired("role"); err != nil {
		panic(err)
	}

	cmd.Version = fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date)
	cmd.SetVersionTemplate("steampipe-config-generator {{.Version}}\n")

	cmd.AddCommand(NewVersionCmd())

	return cmd
}

// finalizeFlags validates the values pflag has already parsed into flags, and fills in the
// defaults and derived fields that used to live in the pre-Cobra flag.Parse() call.
func finalizeFlags(flags *Flags, targetRegions, skipOUs string) error {
	if !slices.Contains(validCredentialSources, flags.CredentialSource) {
		return fmt.Errorf("--credential flag doesn't contain a valid value")
	}

	if !slices.Contains(validImportSchemas, flags.ImportSchema) {
		return fmt.Errorf("--schema flag doesn't contain a valid value")
	}

	if !slices.Contains(validLogFormats, flags.LogFormat) {
		return fmt.Errorf("--log unknown value. Valid values are: default, json")
	}

	if flags.CredentialPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error getting user's home directory: %w", err)
		}
		flags.CredentialPath = filepath.Join(homeDir, ".aws/")
	}

	if flags.ConnectionsPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error getting user's home directory: %w", err)
		}
		flags.ConnectionsPath = filepath.Join(homeDir, ".steampipe/config/")
	}

	if flags.DefaultRegion == "" {
		flags.DefaultRegion = os.Getenv("AWS_REGION")
		if flags.DefaultRegion == "" {
			flags.DefaultRegion = "us-east-1"
			log.Info("default region not defined, using:", flags.DefaultRegion)
		} else {
			log.Debug("default region not defined, using value from env AWS_REGION: ", flags.DefaultRegion)
		}
	}

	if targetRegions == "all" {
		flags.TargetRegions = []string{"*"}
	} else {
		flags.TargetRegions = strings.Split(targetRegions, ",")
	}
	log.Debug("regions: ", flags.TargetRegions)

	flags.SkipOUs = strings.Split(skipOUs, ",")
	log.Debug("skipOUs: ", flags.SkipOUs)

	return nil
}
