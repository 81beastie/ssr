package pipeline

import (
	"github.com/81beastie/ssr/internal/detector"
	"github.com/81beastie/ssr/internal/replacer"
)

type Clipboard interface {
	Copy(text string) error
}

type Pipeline struct {
	clipboard Clipboard
}

func New(clipboard Clipboard) *Pipeline {
	return &Pipeline{clipboard: clipboard}
}

func (p *Pipeline) Run(input string) (string, error) {
	findings := detector.New().Detect(input)
	result := replacer.New().Replace(input, findings)
	if err := p.clipboard.Copy(result); err != nil {
		return "", err
	}
	return result, nil
}
