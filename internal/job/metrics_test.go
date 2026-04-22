package job

import "testing"

func TestMetricsRecordTerminalStatuses(t *testing.T) {
	metrics := &Metrics{}

	metrics.recordTerminalStatus(StatusSuccess)
	metrics.recordTerminalStatus(StatusFailed)
	metrics.recordTerminalStatus(StatusBlocked)

	snapshot := metrics.Snapshot()
	if snapshot.JobsTotal != 3 {
		t.Fatalf("expected jobs_total 3, got %d", snapshot.JobsTotal)
	}
	if snapshot.JobsFailed != 1 {
		t.Fatalf("expected jobs_failed 1, got %d", snapshot.JobsFailed)
	}
	if snapshot.JobsBlocked != 1 {
		t.Fatalf("expected jobs_blocked 1, got %d", snapshot.JobsBlocked)
	}
}
