package main

import (
	"sync"

	"github.com/ashb/jqrepl/jq"
)

type jqInterpreter struct {
	*jq.Jq
	once sync.Once
}

// instances of jq are not thread-safe, so care must be taken to only use one
// instance at a time. create a pool of jqs the size of procs, and fill it with
// initialized jqs.
var jqs = make(chan *jqInterpreter, *procs)

func init() {
	for i := 0; i < *procs; i++ {
		jqVal, err := jq.New()
		if err != nil {
			// most likely OOM?
			fatal("can't initialize jq: %s", err)
		}
		disposeJQ(&jqInterpreter{Jq: jqVal})
	}
}

func disposeJQ(jq *jqInterpreter) {
	jqs <- jq
}

func acquireJQ() *jqInterpreter {
	return <-jqs
}

func AnalyzeJQ(pattern string) AnalyzerFunc {
	return func(b []byte) *Result {
		jqi := acquireJQ()
		defer disposeJQ(jqi)
		var err error
		jqi.once.Do(func() {
			errs := jqi.Compile(pattern, jq.JvArray())
			if len(errs) > 0 {
				// TODO(eric): other errs important here?
				err = errs[0]
			}
		})
		if err != nil {
			fatal("error parsing jq expression: %s", err)
		}
		jv, err := jq.JvFromJSONBytes(b)
		if err != nil {
			return &Result{
				Err: err,
			}
		}
		results, err := jqi.Execute(jv)
		if err != nil {
			return &Result{
				Err: err,
			}
		}
		if len(results) > 0 {
			if results[0].Kind() <= jq.JV_KIND_NULL {
				return nil
			}
			return &Result{
				Match: string(b),
			}
		}
		return nil
	}
}
