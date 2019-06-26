package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
)

var (
	logFile   = flag.String("file", "", "path to log file")
	offset    = flag.Int64("offset", 0, "byte offset to begin at")
	procs     = flag.Int("procs", runtime.NumCPU(), "number of parallel analyzer processes")
	reMatcher = flag.String("re2", "", "RE2 regexp matcher expression")
)

func fatal(formatter string, args ...interface{}) {
	log.Printf(formatter, args...)
	os.Exit(2)
}

func main() {
	flag.Parse()
	if *logFile == "" {
		fatal("-file not specified")
	}
	f, err := os.Open(*logFile)
	if err != nil {
		fatal("couldn't open log file: %s", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			fatal("error closing log file: %s", err)
		}
	}()
	if *offset > 0 {
		if _, err := f.Seek(*offset, io.SeekStart); err != nil {
			fatal("couldn't seek to offset %d: %s", offset, err)
		}
	}
	analyzer := Analyzer{
		Procs: *procs,
		Log:   f,
		Func:  AnalyzeRegexp(*reMatcher),
	}
	profile, err := os.Create("profile.prof")
	if err != nil {
		fatal(err.Error())
	}
	defer profile.Close()
	if err := pprof.StartCPUProfile(profile); err != nil {
		fatal(err.Error())
	}
	defer pprof.StopCPUProfile()
	results := analyzer.Go(context.Background())
	for result := range results {
		fmt.Println(result)
	}
}
