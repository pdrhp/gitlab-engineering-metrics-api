package auth

// Validator handles credential validation
type Validator struct {
	credentials map[string]string
}

// NewValidator creates a new credential validator
func NewValidator(credentials map[string]string) *Validator {
	return &Validator{
		credentials: credentials,
	}
}

// Validate checks if the provided client ID and secret are valid
func (v *Validator) Validate(clientID, clientSecret string) bool {
	if v.credentials == nil {
		return false
	}

	expectedSecret, exists := v.credentials[clientID]
	if !exists {
		return false
	}

	return expectedSecret == clientSecret
}

// ValidateCredentials is a standalone function to validate credentials
func ValidateCredentials(credentials map[string]string, clientID, clientSecret string) bool {
	validator := NewValidator(credentials)
	return validator.Validate(clientID, clientSecret)
}
