package plugins

import (
	"testing"
	"time"

	"ywwzwb/imagespider/models/config"
)

func TestFailureRetryPolicyDefaults(t *testing.T) {
	maxAttempts, retryInterval := failureRetryPolicy(&config.ImageDownloaderConfig{})

	if maxAttempts != 5 {
		t.Fatalf("maxAttempts = %d, want 5", maxAttempts)
	}
	if retryInterval != time.Hour {
		t.Fatalf("retryInterval = %s, want 1h", retryInterval)
	}
}

func TestFailureRetryPolicyUsesConfig(t *testing.T) {
	maxAttempts, retryInterval := failureRetryPolicy(&config.ImageDownloaderConfig{
		FailureRetryMaxCount:  2,
		FailureRetryInterval: 120,
	})

	if maxAttempts != 2 {
		t.Fatalf("maxAttempts = %d, want 2", maxAttempts)
	}
	if retryInterval != 2*time.Minute {
		t.Fatalf("retryInterval = %s, want 2m", retryInterval)
	}
}
