package scanner

import (
	"context"
	"errors"
	"testing"
)

func TestRunScanReturnsContextErrorWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RunScan(ctx, Job{
		RepoURL:  "",
		Branch:   "main",
		ScanType: nil,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRunScanRejectsUnknownScanner(t *testing.T) {
	_, err := RunScan(context.Background(), Job{
		RepoURL:  "",
		Branch:   "main",
		ScanType: []string{"unknown"},
	})
	if err == nil {
		t.Fatal("expected invalid scanner selection to fail")
	}
}
