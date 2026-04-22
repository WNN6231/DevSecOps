package job

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"devsecops-platform/internal/store"
	"devsecops-platform/pkg/common"
)

func TestNormalizeMaxExecutionTimeSec(t *testing.T) {
	value, err := normalizeMaxExecutionTimeSec(0)
	if err != nil {
		t.Fatalf("normalize default timeout: %v", err)
	}
	if value != defaultMaxExecutionTimeSec {
		t.Fatalf("expected default timeout %d, got %d", defaultMaxExecutionTimeSec, value)
	}

	value, err = normalizeMaxExecutionTimeSec(120)
	if err != nil {
		t.Fatalf("normalize explicit timeout: %v", err)
	}
	if value != 120 {
		t.Fatalf("expected explicit timeout 120, got %d", value)
	}

	if _, err := normalizeMaxExecutionTimeSec(-1); !errors.Is(err, ErrInvalidMaxExecutionTime) {
		t.Fatalf("expected ErrInvalidMaxExecutionTime, got %v", err)
	}
}

func TestValidateStatusTransition(t *testing.T) {
	validTransitions := [][2]string{
		{StatusPending, StatusRunning},
		{StatusRunning, StatusSuccess},
		{StatusRunning, StatusFailed},
		{StatusRunning, StatusBlocked},
		{StatusFailed, StatusFailed},
	}

	for _, transition := range validTransitions {
		if err := validateStatusTransition(transition[0], transition[1]); err != nil {
			t.Fatalf("expected transition %s -> %s to be valid: %v", transition[0], transition[1], err)
		}
	}

	invalidTransitions := [][2]string{
		{StatusPending, StatusSuccess},
		{StatusPending, StatusBlocked},
		{StatusSuccess, StatusRunning},
		{StatusFailed, StatusRunning},
	}

	for _, transition := range invalidTransitions {
		if err := validateStatusTransition(transition[0], transition[1]); !errors.Is(err, ErrInvalidJobStatusTransition) {
			t.Fatalf("expected transition %s -> %s to be rejected, got %v", transition[0], transition[1], err)
		}
	}
}

func TestJobExecutionTimedOut(t *testing.T) {
	now := time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC)
	startedAt := now.Add(-2 * time.Minute)

	if !jobExecutionTimedOut(store.ScanJob{
		Status:              StatusRunning,
		StartedAt:           &startedAt,
		MaxExecutionTimeSec: 60,
	}, now) {
		t.Fatal("expected running job past deadline to time out")
	}

	if jobExecutionTimedOut(store.ScanJob{
		Status:              StatusRunning,
		StartedAt:           &startedAt,
		MaxExecutionTimeSec: 300,
	}, now) {
		t.Fatal("expected running job within deadline to remain active")
	}

	if !jobExecutionTimedOut(store.ScanJob{
		Status: StatusRunning,
	}, now) {
		t.Fatal("expected running job without started_at to be treated as timed out")
	}
}

func TestProcessSingleAttemptRecoversPanic(t *testing.T) {
	worker := &Worker{
		service: &Service{
			reportDir: t.TempDir(),
			logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		scanFunc: func(context.Context, Job) ([]common.Finding, error) {
			panic("boom")
		},
	}

	_, _, err := worker.processSingleAttempt(context.Background(), Job{
		ID:                  7,
		MaxExecutionTimeSec: 60,
	})
	if err == nil {
		t.Fatal("expected recovered panic error")
	}
	if !strings.Contains(err.Error(), "worker panic recovered") {
		t.Fatalf("expected recovered panic message, got %v", err)
	}
}
