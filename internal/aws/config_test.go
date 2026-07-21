package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

// fakeSTSAPI is an in-memory stscreds.AssumeRoleAPIClient - no AWS calls happen in these tests.
type fakeSTSAPI struct {
	output *sts.AssumeRoleOutput
	err    error
}

func (f *fakeSTSAPI) AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.output, nil
}

func timePtr(t time.Time) *time.Time { return &t }

func TestAssumeRole(t *testing.T) {
	api := &fakeSTSAPI{
		output: &sts.AssumeRoleOutput{
			Credentials: &types.Credentials{
				AccessKeyId:     strPtr("AKIAFAKE"),
				SecretAccessKey: strPtr("secret"),
				SessionToken:    strPtr("token"),
				Expiration:      timePtr(time.Now().Add(time.Hour)),
			},
		},
	}

	creds, err := assumeRole(t.Context(), api, "arn:aws:iam::111111111111:role/my-role", "test-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AccessKeyID != "AKIAFAKE" {
		t.Errorf("AccessKeyID = %q, want %q", creds.AccessKeyID, "AKIAFAKE")
	}
	if creds.SessionToken != "token" {
		t.Errorf("SessionToken = %q, want %q", creds.SessionToken, "token")
	}
}

func TestAssumeRole_Error(t *testing.T) {
	wantErr := errors.New("AccessDenied")
	api := &fakeSTSAPI{err: wantErr}

	_, err := assumeRole(t.Context(), api, "arn:aws:iam::111111111111:role/my-role", "test-session")
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestWithAssumedRole(t *testing.T) {
	api := &fakeSTSAPI{
		output: &sts.AssumeRoleOutput{
			Credentials: &types.Credentials{
				AccessKeyId:     strPtr("AKIAFAKE"),
				SecretAccessKey: strPtr("secret"),
				SessionToken:    strPtr("token"),
				Expiration:      timePtr(time.Now().Add(time.Hour)),
			},
		},
	}

	awscfg, err := withAssumedRole(t.Context(), api, Config{
		AssumeRoleArn: "arn:aws:iam::111111111111:role/my-role",
		Region:        "us-east-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if awscfg.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", awscfg.Region, "us-east-1")
	}

	creds, err := awscfg.Credentials.Retrieve(t.Context())
	if err != nil {
		t.Fatalf("retrieving credentials from resulting config: %v", err)
	}
	if creds.AccessKeyID != "AKIAFAKE" {
		t.Errorf("AccessKeyID = %q, want %q", creds.AccessKeyID, "AKIAFAKE")
	}
}

func TestWithAssumedRole_AssumeRoleError(t *testing.T) {
	wantErr := errors.New("AccessDenied")
	api := &fakeSTSAPI{err: wantErr}

	_, err := withAssumedRole(t.Context(), api, Config{AssumeRoleArn: "arn:aws:iam::111111111111:role/my-role"})
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want it to wrap %v", err, wantErr)
	}
}
