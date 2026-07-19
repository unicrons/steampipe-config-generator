package cmd_test

import (
	"bytes"
	"testing"

	"github.com/unicrons/steampipe-config-generator/cmd"
)

// execute runs cmd with the given args against a fresh command tree and returns its output
// and error. run is invoked only if flag parsing/validation succeeds.
func execute(t *testing.T, run func(*cmd.Flags) error, args ...string) (string, error) {
	t.Helper()

	root := cmd.NewRootCmd(run)
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs(args)

	err := root.Execute()
	return out.String(), err
}

func TestNewRootCmd_HappyPath(t *testing.T) {
	var got *cmd.Flags
	run := func(f *cmd.Flags) error {
		got = f
		return nil
	}

	_, err := execute(t, run,
		"--role", "my-role",
		"--credential", "Ec2InstanceMetadata",
		"--path", "/tmp/aws",
		"--connections", "/tmp/steampipe",
		"--schema", "disabled",
		"--region", "eu-west-1",
		"--regions", "eu-west-1,us-east-1",
		"--assume", "arn:aws:iam::123456789012:role/assume-me",
		"--log", "json",
		"--skipOUs", "ou-1,ou-2",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("run was not called")
	}
	if got.RoleName != "my-role" {
		t.Errorf("RoleName = %q, want %q", got.RoleName, "my-role")
	}
	if got.CredentialSource != "Ec2InstanceMetadata" {
		t.Errorf("CredentialSource = %q, want %q", got.CredentialSource, "Ec2InstanceMetadata")
	}
	if got.ImportSchema != "disabled" {
		t.Errorf("ImportSchema = %q, want %q", got.ImportSchema, "disabled")
	}
	wantRegions := []string{"eu-west-1", "us-east-1"}
	if len(got.TargetRegions) != len(wantRegions) || got.TargetRegions[0] != wantRegions[0] || got.TargetRegions[1] != wantRegions[1] {
		t.Errorf("TargetRegions = %v, want %v", got.TargetRegions, wantRegions)
	}
	wantSkipOUs := []string{"ou-1", "ou-2"}
	if len(got.SkipOUs) != len(wantSkipOUs) || got.SkipOUs[0] != wantSkipOUs[0] || got.SkipOUs[1] != wantSkipOUs[1] {
		t.Errorf("SkipOUs = %v, want %v", got.SkipOUs, wantSkipOUs)
	}
}

func TestNewRootCmd_HappyPath_Defaults(t *testing.T) {
	var got *cmd.Flags
	run := func(f *cmd.Flags) error {
		got = f
		return nil
	}

	_, err := execute(t, run, "--role", "my-role")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.CredentialSource != "Environment" {
		t.Errorf("CredentialSource default = %q, want %q", got.CredentialSource, "Environment")
	}
	if got.ImportSchema != "enabled" {
		t.Errorf("ImportSchema default = %q, want %q", got.ImportSchema, "enabled")
	}
	if got.LogFormat != "default" {
		t.Errorf("LogFormat default = %q, want %q", got.LogFormat, "default")
	}
	if len(got.TargetRegions) != 1 || got.TargetRegions[0] != "*" {
		t.Errorf("TargetRegions default = %v, want [*]", got.TargetRegions)
	}
}

func TestNewRootCmd_RoleRequired(t *testing.T) {
	run := func(*cmd.Flags) error {
		t.Fatal("run should not be called when --role is missing")
		return nil
	}

	_, err := execute(t, run)
	if err == nil {
		t.Fatal("expected an error when --role is missing")
	}
}

func TestNewRootCmd_InvalidFlagValues(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "invalid credential",
			args: []string{"--role", "x", "--credential", "Bogus"},
		},
		{
			name: "invalid schema",
			args: []string{"--role", "x", "--schema", "Bogus"},
		},
		{
			name: "invalid log format",
			args: []string{"--role", "x", "--log", "Bogus"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := func(*cmd.Flags) error {
				t.Fatal("run should not be called for an invalid flag value")
				return nil
			}

			_, err := execute(t, run, tt.args...)
			if err == nil {
				t.Fatalf("expected an error for args %v", tt.args)
			}
		})
	}
}

func TestNewRootCmd_Version(t *testing.T) {
	run := func(*cmd.Flags) error {
		t.Fatal("run should not be called for --version")
		return nil
	}

	out, err := execute(t, run, "--version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == "" {
		t.Fatal("expected non-empty output for --version")
	}
}

func TestNewVersionCmd(t *testing.T) {
	run := func(*cmd.Flags) error {
		t.Fatal("run should not be called for the version subcommand")
		return nil
	}

	out, err := execute(t, run, "version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == "" {
		t.Fatal("expected non-empty output for the version subcommand")
	}
}
