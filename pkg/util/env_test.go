package util

import (
	"os"
	"testing"
)

func TestGetEnvWithDefault(t *testing.T) {
	testCases := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "Environment variable is set",
			key:          "TEST_ENV_VAR_1",
			defaultValue: "default_value",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "Environment variable is not set",
			key:          "TEST_ENV_VAR_2",
			defaultValue: "default_value",
			envValue:     "",
			expected:     "default_value",
		},
		{
			name:         "Environment variable is set to empty string",
			key:          "TEST_ENV_VAR_3",
			defaultValue: "default_value",
			envValue:     "",
			expected:     "default_value",
		},
		{
			name:         "Default value is empty string",
			key:          "TEST_ENV_VAR_4",
			defaultValue: "",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "Both values are empty strings",
			key:          "TEST_ENV_VAR_5",
			defaultValue: "",
			envValue:     "",
			expected:     "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv(tc.key, tc.envValue)
				defer os.Unsetenv(tc.key)
			} else {
				os.Unsetenv(tc.key)
			}

			result := GetEnvWithDefault(tc.key, tc.defaultValue)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}
