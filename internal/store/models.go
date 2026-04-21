package store

import "time"

type ScanJob struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	RepoURL     string     `gorm:"column:repo_url;type:varchar(512);not null"`
	Branch      string     `gorm:"column:branch;type:varchar(255);not null"`
	ScanType    string     `gorm:"column:scan_type;type:text;not null"`
	Status      string     `gorm:"column:status;type:varchar(32);not null"`
	BlockOnHigh bool       `gorm:"column:block_on_high;not null;default:false"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	StartedAt   *time.Time `gorm:"column:started_at"`
	FinishedAt  *time.Time `gorm:"column:finished_at"`
}

func (ScanJob) TableName() string {
	return "scan_jobs"
}

type ScanResult struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	JobID          uint64    `gorm:"column:job_id;not null;index"`
	ScannerName    string    `gorm:"column:scanner_name;type:varchar(64);not null"`
	Severity       string    `gorm:"column:severity;type:varchar(16);not null"`
	RuleID         string    `gorm:"column:rule_id;type:varchar(128);not null"`
	FilePath       string    `gorm:"column:file_path;type:varchar(1024);not null"`
	LineNumber     int       `gorm:"column:line_number;not null"`
	Title          string    `gorm:"column:title;type:varchar(255);not null"`
	Description    string    `gorm:"column:description;type:text;not null"`
	Evidence       string    `gorm:"column:evidence;type:text;not null"`
	Recommendation string    `gorm:"column:recommendation;type:text;not null"`
	Hash           string    `gorm:"column:hash;type:varchar(128);not null;index"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (ScanResult) TableName() string {
	return "scan_results"
}

type ScanReport struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	JobID       uint64    `gorm:"column:job_id;not null;uniqueIndex"`
	ReportPath  string    `gorm:"column:report_path;type:varchar(1024);not null"`
	SummaryJSON string    `gorm:"column:summary_json;type:text;not null"`
	HighCount   int       `gorm:"column:high_count;not null"`
	MediumCount int       `gorm:"column:medium_count;not null"`
	LowCount    int       `gorm:"column:low_count;not null"`
	RiskScore   int       `gorm:"column:risk_score;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (ScanReport) TableName() string {
	return "scan_reports"
}
