package store

import (
	"context"

	"gorm.io/gorm"
)

func ListScanResults(ctx context.Context, db *gorm.DB, jobID int64, offset, limit int) ([]ScanResult, int64, error) {
	var total int64
	baseQuery := db.WithContext(ctx).Model(&ScanResult{}).Where("job_id = ?", jobID)

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	results := make([]ScanResult, 0, limit)
	if err := baseQuery.
		Order("id ASC").
		Offset(offset).
		Limit(limit).
		Find(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}
