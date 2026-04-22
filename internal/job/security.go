package job

import (
	"errors"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	branchPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,255}$`)
	gitSSHPattern = regexp.MustCompile(`^git@[A-Za-z0-9.-]+:[A-Za-z0-9._/-]+(?:\.git)?$`)
)

func sanitizeCreateJobInput(repoURL, branch string, scanType []string) (string, string, []string, error) {
	sanitizedRepoURL, err := sanitizeRepoURL(repoURL)
	if err != nil {
		return "", "", nil, err
	}

	sanitizedBranch, err := sanitizeBranch(branch)
	if err != nil {
		return "", "", nil, err
	}

	sanitizedScanType, err := sanitizeScanTypes(scanType)
	if err != nil {
		return "", "", nil, err
	}

	return sanitizedRepoURL, sanitizedBranch, sanitizedScanType, nil
}

func sanitizeRepoURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrInvalidRepoURL
	}
	if strings.ContainsRune(value, '\x00') {
		return "", ErrInvalidRepoURL
	}

	switch {
	case strings.Contains(value, "://"):
		u, err := url.Parse(value)
		if err != nil {
			return "", ErrInvalidRepoURL
		}
		if u.Host == "" {
			return "", ErrInvalidRepoURL
		}
		switch u.Scheme {
		case "http", "https", "ssh":
		default:
			return "", ErrInvalidRepoURL
		}
		if u.User != nil || u.RawQuery != "" || u.Fragment != "" || hasTraversalSegment(u.Path) {
			return "", ErrInvalidRepoURL
		}

		u.Path = path.Clean(u.Path)
		return u.String(), nil
	case strings.HasPrefix(value, "git@"):
		parts := strings.SplitN(value, ":", 2)
		if len(parts) != 2 || !gitSSHPattern.MatchString(value) || hasTraversalSegment(parts[1]) {
			return "", ErrInvalidRepoURL
		}
		return value, nil
	default:
		if hasTraversalSegment(value) {
			return "", ErrInvalidRepoURL
		}

		return filepath.Clean(value), nil
	}
}

func sanitizeBranch(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") || hasTraversalSegment(value) || !branchPattern.MatchString(value) {
		return "", ErrInvalidBranch
	}

	return value, nil
}

func sanitizeScanTypes(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{}, nil
	}

	allowed := map[string]struct{}{
		"sast":   {},
		"secret": {},
		"sca":    {},
		"dast":   {},
	}

	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			return nil, ErrInvalidScanType
		}
		if _, exists := seen[value]; exists {
			return nil, ErrInvalidScanType
		}

		seen[value] = struct{}{}
		out = append(out, value)
	}

	return out, nil
}

func redactRepoURLForLog(value string) string {
	if value == "" {
		return ""
	}

	if strings.Contains(value, "://") {
		u, err := url.Parse(value)
		if err != nil {
			return "[invalid-repo-url]"
		}

		u.User = nil
		u.RawQuery = ""
		u.Fragment = ""
		return u.String()
	}

	if strings.HasPrefix(value, "git@") {
		return value
	}

	return filepath.Base(filepath.Clean(value))
}

func hasTraversalSegment(value string) bool {
	segments := strings.FieldsFunc(value, func(r rune) bool {
		return r == '/' || r == '\\'
	})

	for _, segment := range segments {
		if segment == ".." {
			return true
		}
	}

	return false
}

func isInputValidationError(err error) bool {
	return errors.Is(err, ErrInvalidRepoURL) || errors.Is(err, ErrInvalidBranch) || errors.Is(err, ErrInvalidScanType) || errors.Is(err, ErrInvalidMaxExecutionTime)
}
