package fingerprint

import "testing"

func TestEvent(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		stacktrace string
		expected   string
	}{
		{
			name:       "basic event",
			message:    "error",
			stacktrace: "something went wrong",
			expected:   "2f63f8e1fe56595e51ef448ca31708e719c880b0d0d6ca98189dd933cbddf07a",
		},
		{
			name:       "different event",
			message:    "error",
			stacktrace: "another error occurred",
			expected:   "efa5526732185c1ea8858358658e3c170cf93ccc4a721a59e43f43b42ee0f936",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fingerprint := Event(tt.message, tt.stacktrace)
			if fingerprint != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, fingerprint)
			}
		})
	}
}
