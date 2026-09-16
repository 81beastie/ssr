package replacer

import (
	"sort"
	"strconv"
	"strings"

	"github.com/81beastie/ssr/internal/detector"
)

type Replacer struct{}

func New() *Replacer {
	return &Replacer{}
}

func (r *Replacer) Replace(text string, findings []detector.Finding) string {
	if len(findings) == 0 {
		return text
	}

	sorted := make([]detector.Finding, len(findings))
	copy(sorted, findings)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })

	numbers := numberDistinctValues(sorted)
	placeholders := placeholderPerValue(sorted, numbers)

	var sb strings.Builder
	last := 0
	for _, f := range sorted {
		sb.WriteString(text[last:f.Start])
		sb.WriteString(placeholders[f.Value])
		last = f.End
	}
	sb.WriteString(text[last:])
	return sb.String()
}

func numberDistinctValues(sorted []detector.Finding) map[string]int {
	distinct := map[string][]string{}
	seen := map[string]bool{}
	for _, f := range sorted {
		key := f.Type + "\x00" + f.Value
		if !seen[key] {
			seen[key] = true
			distinct[f.Type] = append(distinct[f.Type], f.Value)
		}
	}

	numbers := map[string]int{}
	for secretType, values := range distinct {
		if len(values) < 2 {
			continue
		}
		for i, v := range values {
			numbers[secretType+"\x00"+v] = i + 1
		}
	}
	return numbers
}

func placeholderPerValue(sorted []detector.Finding, numbers map[string]int) map[string]string {
	placeholders := map[string]string{}
	for _, f := range sorted {
		key := f.Type + "\x00" + f.Value
		if n, ok := numbers[key]; ok {
			placeholders[f.Value] = "<ssr " + f.Type + ":" + strconv.Itoa(n) + ">"
			continue
		}
		placeholders[f.Value] = "<ssr " + f.Type + ">"
	}
	return placeholders
}
