package core

import (
	"fmt"
	"regexp"
	"strings"
)

// Rule is saved independently of the single currently selected rule.
type Rule struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
}

func (r Rule) Compile() (*regexp.Regexp, error) {
	if strings.TrimSpace(r.Name) == "" {
		return nil, fmt.Errorf("rule name is empty")
	}
	if r.Pattern == "" {
		return nil, fmt.Errorf("pattern is empty")
	}
	if _, err := regexp.Compile(r.Pattern); err != nil {
		return nil, err
	}
	// Absolute anchors enforce whole-filename matching even with (?m) or |.
	return regexp.Compile(`\A(?:` + r.Pattern + `)\z`)
}
