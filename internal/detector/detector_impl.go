package detector

import "regexp"

type Finding struct {
	Type  string
	Value string
	Start int
	End   int
}

type Detector struct{}

type rule struct {
	secretType string
	pattern    *regexp.Regexp
}

var rules = []rule{
	{secretType: "token", pattern: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)},
	{secretType: "token", pattern: regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{36,255}\b`)},
	{secretType: "token", pattern: regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{secretType: "token", pattern: regexp.MustCompile(`(?i)\bbearer\s+([A-Za-z0-9\-._~+/]+=*)`)},
	{secretType: "password", pattern: regexp.MustCompile(`[a-z][a-z0-9+.-]*://[^:/@\s]+:([^@\s]+)@`)},
	{secretType: "password", pattern: regexp.MustCompile(`(?i)\bpassword\s*[=:]\s*([^\s"']+)`)},
	{secretType: "token", pattern: regexp.MustCompile(`\b[A-Z0-9_]*TOKEN[A-Z0-9_]*=([A-Za-z0-9_\-]+)`)},
}

func New() *Detector {
	return &Detector{}
}

func (d *Detector) Detect(text string) []Finding {
	findings := []Finding{}
	for _, r := range rules {
		for _, loc := range r.pattern.FindAllStringSubmatchIndex(text, -1) {
			valueStart, valueEnd := loc[0], loc[1]
			if len(loc) > 2 {
				valueStart, valueEnd = loc[2], loc[3]
			}
			if overlaps(findings, valueStart, valueEnd) {
				continue
			}
			findings = append(findings, Finding{
				Type:  r.secretType,
				Value: text[valueStart:valueEnd],
				Start: valueStart,
				End:   valueEnd,
			})
		}
	}
	return findings
}

func overlaps(findings []Finding, start, end int) bool {
	for _, f := range findings {
		if start < f.End && f.Start < end {
			return true
		}
	}
	return false
}
