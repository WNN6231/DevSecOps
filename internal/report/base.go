package report

import (
	"errors"
	"fmt"
	"path/filepath"
)

var ErrReportNotFound = errors.New("report not found")
var ErrInvalidReportDir = errors.New("report dir must be absolute")

func reportPath(reportDir, filename string) (string, error) {
	if !filepath.IsAbs(reportDir) {
		return "", ErrInvalidReportDir
	}

	return filepath.Join(filepath.Clean(reportDir), filename), nil
}

func markdownFilename(jobID int64) string {
	return fmt.Sprintf("%d.md", jobID)
}
