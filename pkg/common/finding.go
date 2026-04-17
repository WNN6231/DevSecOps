package common

type Finding struct {
	Scanner        string
	Severity       string
	RuleID         string
	Title          string
	Description    string
	FilePath       string
	LineNumber     int
	Evidence       string
	Recommendation string
	Hash           string
}
