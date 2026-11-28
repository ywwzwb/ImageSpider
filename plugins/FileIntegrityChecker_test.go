package plugins

import "testing"

func TestNotifyNewImageDoesNotBlock(t *testing.T) {
	checker := newFileIntegrityChecker()

	checker.NotifyNewImage("yande_re")
	select {
	case <-checker.wakeUp:
	default:
		t.Fatal("expected a wake-up signal")
	}

	// channel is full now; a second notification must not block or panic
	checker.NotifyNewImage("yande_re")
}
