package main

import "regexp"

func AnalyzeRegexp(pattern string) AnalyzerFunc {
	re, err := regexp.Compile(pattern)
	if err != nil {
		fatal("invalid regexp: %s", err)
	}
	return func(b []byte) *Result {
		if re.Match(b) {
			return &Result{Match: string(b)}
		}
		return nil
	}
}
