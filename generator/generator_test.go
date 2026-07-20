package generator

import "testing"

func TestNew(t *testing.T) {
	// LoadConfig without AssumeRoleArn only resolves the local SDK config chain - it makes no
	// network calls, so this succeeds even without real AWS credentials in the environment.
	g, err := New(t.Context(), Options{RoleName: "my-role"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g == nil {
		t.Fatal("New returned a nil Generator")
	}
}
