package aws

import (
	"testing"
)

func TestSetLogLevel(t *testing.T) {
	tests := []struct {
		name       string
		level      string
		shouldFail bool
	}{
		{
			name:       "valid debug level",
			level:      "debug",
			shouldFail: false,
		},
		{
			name:       "valid info level",
			level:      "info",
			shouldFail: false,
		},
		{
			name:       "valid warn level",
			level:      "warn",
			shouldFail: false,
		},
		{
			name:       "valid error level",
			level:      "error",
			shouldFail: false,
		},
		{
			name:       "invalid level",
			level:      "invalid",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: SetLogLevel calls log.Fatalf on error, which will exit the test
			// In a real implementation, we might want to refactor SetLogLevel to return an error
			// For now, we only test valid levels
			if tt.shouldFail {
				t.Skip("SetLogLevel calls Fatal on invalid level; skipping failure case")
			}
			SetLogLevel(tt.level)
		})
	}
}

func TestNewConfigurationManagerDefaults(t *testing.T) {
	// This test is environment-dependent and will pass if valid AWS credentials
	// are available in the environment (which is expected in most test environments)
	// For a proper test, we would need to mock the credential chain or use test doubles

	// Minimal smoke test: verify that NewConfigurationManager doesn't crash
	cm, err := NewConfigurationManager()

	// The behavior depends on the environment:
	// - If credentials are available, cm should be non-nil and err should be nil
	// - If credentials are not available, cm should be nil and err should be non-nil

	if err == nil && cm == nil {
		t.Error("NewConfigurationManager() returned both nil cm and nil error")
	}

	if err != nil && cm != nil {
		t.Error("NewConfigurationManager() returned both cm and error")
	}
}

func TestDefaultAssumeRoleConstant(t *testing.T) {
	if DefaultAssumeRole != "terraform" {
		t.Errorf("DefaultAssumeRole = %q, want %q", DefaultAssumeRole, "terraform")
	}
}

func TestProfileStringConstant(t *testing.T) {
	if ProfileString != "AWS_PROFILE" {
		t.Errorf("ProfileString = %q, want %q", ProfileString, "AWS_PROFILE")
	}
}

func TestBuildCredentialHint(t *testing.T) {
	err := buildCredentialHint()
	if err == nil {
		t.Error("buildCredentialHint() returned nil, expected error")
	}

	errMsg := err.Error()
	expectedStrings := []string{
		"credential/region resolution failed",
		"AWS_ACCESS_KEY_ID",
		"AWS_PROFILE",
		"aws sso login",
		"IMDS/IRSA",
	}

	for _, expected := range expectedStrings {
		if !contains(errMsg, expected) {
			t.Errorf("buildCredentialHint() error message missing expected string: %q", expected)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
