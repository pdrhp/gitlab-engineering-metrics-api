package auth

import (
	"testing"
)

func TestValidateCredentials(t *testing.T) {
	// Setup test credentials
	creds := map[string]string{
		"client1": "secret1",
		"client2": "secret2",
	}

	tests := []struct {
		name      string
		clientID  string
		secret    string
		wantValid bool
	}{
		{
			name:      "valid credentials",
			clientID:  "client1",
			secret:    "secret1",
			wantValid: true,
		},
		{
			name:      "invalid client id",
			clientID:  "unknown",
			secret:    "secret1",
			wantValid: false,
		},
		{
			name:      "invalid secret",
			clientID:  "client1",
			secret:    "wrongsecret",
			wantValid: false,
		},
		{
			name:      "empty client id",
			clientID:  "",
			secret:    "secret1",
			wantValid: false,
		},
		{
			name:      "empty secret",
			clientID:  "client1",
			secret:    "",
			wantValid: false,
		},
		{
			name:      "both empty",
			clientID:  "",
			secret:    "",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateCredentials(creds, tt.clientID, tt.secret)
			if got != tt.wantValid {
				t.Errorf("ValidateCredentials() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestValidator(t *testing.T) {
	creds := map[string]string{
		"client1": "secret1",
	}

	validator := NewValidator(creds)

	tests := []struct {
		name      string
		clientID  string
		secret    string
		wantValid bool
	}{
		{
			name:      "valid credentials via validator",
			clientID:  "client1",
			secret:    "secret1",
			wantValid: true,
		},
		{
			name:      "invalid credentials via validator",
			clientID:  "client1",
			secret:    "wrong",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.Validate(tt.clientID, tt.secret)
			if got != tt.wantValid {
				t.Errorf("Validator.Validate() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}
