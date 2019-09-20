package mail

import (
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		mail     string
		expected bool
	}{
		{
			mail:     "user@example.com",
			expected: true,
		},
		{
			mail:     "user_example.com",
			expected: false,
		},
		{
			mail:     "user_@example.com",
			expected: true,
		},
		{
			mail:     "user#@example.com",
			expected: false,
		},
	}

	for _, tt := range tests {

		result := Validate(tt.mail)
		if result != tt.expected {
			t.Errorf("TestValidate()\n got1 = %v, \nwant %v", result, tt.expected)
		}
	}
}
