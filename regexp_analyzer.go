package main

import (
	"regexp"
)

func AnalyzeRegexp(pattern string) AnalyzerFunc {
	re, err := regexp.Compile(pattern)
	if err != nil {
		fatal("invalid regexp: %s", err)
	}
	return func(b []byte) *Result {
		inverse := plugin.InverseMatch
		if inverse && !re.Match(b) {
			return &Result{Match: string(b), Inverse: inverse}
		}
		if !inverse && re.Match(b) {
			return &Result{Match: string(b), Inverse: inverse}
		}
		if re.Match(b) {
		} else {
		}
		return nil
	}
}
