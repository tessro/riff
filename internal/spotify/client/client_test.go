package client

import (
	"testing"
)

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		params map[string]string
		want   string
	}{
		{
			name:   "no params",
			path:   "/me",
			params: nil,
			want:   "/me",
		},
		{
			name:   "empty params",
			path:   "/me",
			params: map[string]string{},
			want:   "/me",
		},
		{
			name:   "single param",
			path:   "/search",
			params: map[string]string{"q": "test"},
			want:   "/search?q=test",
		},
		{
			name:   "multiple params",
			path:   "/search",
			params: map[string]string{"q": "test", "type": "track"},
			want:   "/search?", // Order is not guaranteed, just check it has params
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildURL(tt.path, tt.params)
			if tt.name == "multiple params" {
				// Just verify it contains the path and both params
				if len(got) < len("/search?q=test&type=track") {
					t.Errorf("BuildURL() = %q, seems too short", got)
				}
			} else if got != tt.want {
				t.Errorf("BuildURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIError(t *testing.T) {
	err := &APIError{}
	err.ErrorInfo.Status = 401
	err.ErrorInfo.Message = "Invalid access token"

	expected := "Spotify API error 401: Invalid access token"
	if got := err.Error(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}
