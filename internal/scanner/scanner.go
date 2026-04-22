package scanner

import (
	"context"
	"fmt"

	"devsecops-platform/internal/scanner/dast"
	"devsecops-platform/internal/scanner/sast"
	"devsecops-platform/internal/scanner/sca"
	"devsecops-platform/internal/scanner/secret"
	"devsecops-platform/pkg/common"
)

// Scanner is the common interface every scan engine must implement.
type Scanner interface {
	Scan(ctx context.Context, repoURL, branch string) ([]common.Finding, error)
}

// Job is the minimal view of a scan job the runner needs.
type Job struct {
	RepoURL  string
	Branch   string
	ScanType []string // "sast" | "secret" | "sca" | "dast"; empty means all
}

// RunScan orchestrates all requested scanner types and returns aggregated findings.
// It is the single entry point shared by the API worker and the CLI.
func RunScan(ctx context.Context, job Job) ([]common.Finding, error) {
	enabled := resolveEnabled(job.ScanType)

	registry := map[string]Scanner{
		"sast":   sast.NewScanner(),
		"secret": secret.NewScanner(),
		"sca":    sca.NewScanner(),
		"dast":   dast.NewScanner(),
	}

	var all []common.Finding
	for _, name := range enabled {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		s, ok := registry[name]
		if !ok {
			return nil, fmt.Errorf("unknown scanner: %s", name)
		}

		findings, err := s.Scan(ctx, job.RepoURL, job.Branch)
		if err != nil {
			return nil, fmt.Errorf("%s scanner: %w", name, err)
		}

		all = append(all, findings...)
	}

	return all, nil
}

// EnabledScanners returns the scanner names that RunScan will invoke for a given job.
// Callers use this to populate report metadata without re-running scans.
func EnabledScanners(scanType []string) []string {
	return resolveEnabled(scanType)
}

var allScanners = []string{"sast", "secret", "sca", "dast"}

func resolveEnabled(scanType []string) []string {
	if len(scanType) == 0 {
		out := make([]string, len(allScanners))
		copy(out, allScanners)
		return out
	}

	return append([]string(nil), scanType...)
}
