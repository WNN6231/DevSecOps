package job

import "sync/atomic"

type Metrics struct {
	jobsTotal   atomic.Int64
	jobsFailed  atomic.Int64
	jobsBlocked atomic.Int64
}

type MetricsSnapshot struct {
	JobsTotal   int64 `json:"jobs_total"`
	JobsFailed  int64 `json:"jobs_failed"`
	JobsBlocked int64 `json:"jobs_blocked"`
}

func (m *Metrics) recordTerminalStatus(status string) {
	m.jobsTotal.Add(1)

	switch status {
	case StatusFailed:
		m.jobsFailed.Add(1)
	case StatusBlocked:
		m.jobsBlocked.Add(1)
	}
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		JobsTotal:   m.jobsTotal.Load(),
		JobsFailed:  m.jobsFailed.Load(),
		JobsBlocked: m.jobsBlocked.Load(),
	}
}
