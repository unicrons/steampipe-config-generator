package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unicrons/steampipe-config-generator/cmd"
	"github.com/unicrons/steampipe-config-generator/generator"
)

// fakeGenerator is an in-memory generator.Generator - no AWS calls happen in these tests.
type fakeGenerator struct {
	accounts []generator.Account
	err      error
}

func (f *fakeGenerator) Accounts(ctx context.Context) ([]generator.Account, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.accounts, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	fake := &fakeGenerator{accounts: []generator.Account{
		{
			Name:             "team_foo",
			RoleARN:          "arn:aws:iam::111111111111:role/my-role",
			CredentialSource: "Environment",
			DefaultRegion:    "us-east-1",
			ImportSchema:     "enabled",
			TargetRegions:    []string{"*"},
		},
	}}
	newGenerator := func(ctx context.Context, opts generator.Options) (generator.Generator, error) {
		return fake, nil
	}

	flags := &cmd.Flags{
		CredentialPath:  filepath.Join(dir, "creds"),
		ConnectionsPath: filepath.Join(dir, "conn"),
	}

	if err := run(t.Context(), discardLogger(), flags, newGenerator); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(flags.CredentialPath, "credentials")); err != nil {
		t.Errorf("expected credentials file to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(flags.ConnectionsPath, "aws.spc")); err != nil {
		t.Errorf("expected connections file to exist: %v", err)
	}
}

func TestRun_NewGeneratorError(t *testing.T) {
	wantErr := errors.New("boom")
	newGenerator := func(ctx context.Context, opts generator.Options) (generator.Generator, error) {
		return nil, wantErr
	}

	err := run(t.Context(), discardLogger(), &cmd.Flags{}, newGenerator)
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestRun_AccountsError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeGenerator{err: wantErr}
	newGenerator := func(ctx context.Context, opts generator.Options) (generator.Generator, error) {
		return fake, nil
	}

	err := run(t.Context(), discardLogger(), &cmd.Flags{}, newGenerator)
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestWriteCredentialsFile(t *testing.T) {
	dir := t.TempDir()
	accounts := []generator.Account{
		{Name: "team_foo", RoleARN: "arn:aws:iam::111111111111:role/my-role", CredentialSource: "Environment"},
	}

	if err := writeCredentialsFile(dir, accounts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "credentials"))
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}
	if !strings.Contains(string(got), "[team_foo]") {
		t.Errorf("generated file missing expected content, got:\n%s", got)
	}
}

func TestWriteCredentialsFile_CreatesPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "aws")

	if err := writeCredentialsFile(dir, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "credentials")); err != nil {
		t.Errorf("expected credentials file to exist: %v", err)
	}
}

func TestWriteConnectionsFile(t *testing.T) {
	dir := t.TempDir()
	accounts := []generator.Account{
		{Name: "team_foo", DefaultRegion: "us-east-1", ImportSchema: "enabled", TargetRegions: []string{"*"}},
	}

	if err := writeConnectionsFile(dir, "", accounts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "aws.spc"))
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}
	if !strings.Contains(string(got), `connection "aws_team_foo"`) {
		t.Errorf("generated file missing expected content, got:\n%s", got)
	}
}

func TestWriteConnectionsFile_InvalidTemplatePath(t *testing.T) {
	err := writeConnectionsFile(t.TempDir(), "/no/such/template.tmpl", nil)
	if err == nil {
		t.Fatal("expected an error for a nonexistent template path")
	}
}
