package store

import (
	"sync"
	"time"
)

type JobRecord struct {
	ID          int64
	RepoURL     string
	Branch      string
	ScanType    []string
	BlockOnHigh bool
	Status      string
	CreatedAt   time.Time
}

type JobStore struct {
	mu     sync.RWMutex
	nextID int64
	jobs   map[int64]JobRecord
}

func NewJobStore() *JobStore {
	return &JobStore{
		nextID: 1,
		jobs:   make(map[int64]JobRecord),
	}
}

func (s *JobStore) Create(record JobRecord) (JobRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record.ID = s.nextID
	record.ScanType = append([]string(nil), record.ScanType...)
	s.jobs[record.ID] = record
	s.nextID++

	return record, nil
}

func (s *JobStore) GetByID(id int64) (JobRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.jobs[id]
	if !ok {
		return JobRecord{}, false
	}

	record.ScanType = append([]string(nil), record.ScanType...)
	return record, true
}
