package main

import (
	"reflect"
	"testing"
)

func TestJQAnalyzer(t *testing.T) {
	tests := []struct {
		Pattern    string
		LogLines   [][]byte
		ExpResults []*Result
	}{
		{
			Pattern: `.foo`,
			LogLines: [][]byte{
				[]byte(`{"foo": "bar"}`),
				[]byte(`{"bar": "foo"}`),
			},
			ExpResults: []*Result{
				&Result{
					Match: `{"foo": "bar"}`,
				},
				nil,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Pattern, func(t *testing.T) {
			analyzer := AnalyzeJQ(test.Pattern)
			results := make([]*Result, 0)
			for _, line := range test.LogLines {
				results = append(results, analyzer(line))
			}
			if got, want := len(results), len(test.ExpResults); got != want {
				t.Fatal("wrong number of results")
			}
			for i := range results {
				if got, want := results[i], test.ExpResults[i]; !reflect.DeepEqual(got, want) {
					t.Fatalf("bad result %d: got %v, want %v", i, got, want)
				}
			}
		})
	}
}
