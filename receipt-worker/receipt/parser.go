package receipt

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Parser interface {
	Detect(lines []string) bool
	Parse(lines []string, r *Receipt) (*Receipt, error)
}

var parsers []Parser

func Register(p Parser) {
	parsers = append(parsers, p)
}

func Parse(raw string) (*Receipt, error) {
	lines := strings.Split(raw, "\n")
	for _, p := range parsers {
		if p.Detect(lines) {
			return p.Parse(lines, &Receipt{})
		}
	}
	return nil, fmt.Errorf("no parser found for receipt")
}

func parseFloat(s string) float64 {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return math.Round(f*100) / 100
}
