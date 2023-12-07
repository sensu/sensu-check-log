package main

import (
	"bufio"
	"context"
	"io"
	"sync"
	"sync/atomic"
)

const bufSize = 1000

type Analyzer struct {
	Procs          int
	Path           string
	Log            io.Reader
	Func           AnalyzerFunc
	Offset         int64
	wg             sync.WaitGroup
	bytesRead      int64
	VerboseResults bool
}

type discardInterface interface {
	io.Writer
	io.ReaderFrom
}

type DiscardWriter struct {
	discardWriter discardInterface
	BytesRead     int64
}

func (d *DiscardWriter) Write(b []byte) (int, error) {
	n, err := d.discardWriter.Write(b)
	d.BytesRead += int64(n)
	return n, err
}

func (d *DiscardWriter) ReadFrom(r io.Reader) (int64, error) {
	n, err := d.discardWriter.ReadFrom(r)
	d.BytesRead += n
	return n, err
}

func NewDiscardWriter() *DiscardWriter {
	discard := io.Discard.(discardInterface)
	return &DiscardWriter{discardWriter: discard}
}

type AnalyzerFunc func([]byte) *Result

type Result struct {
	Path   string `json:"path"`
	Match  string `json:"match,omitempty"`
	Err    error  `json:"error,omitempty"`
	Offset int64  `json:"offset"`
}

type LineMsg struct {
	Line   []byte
	Offset int64
}

func (a *Analyzer) Go(ctx context.Context) <-chan Result {
	resultC := make(chan Result)
	a.wg.Add(a.Procs)
	producer := a.startProducer(ctx)
	go func() {
		a.wg.Wait()
		close(resultC)
	}()
	for i := 0; i < a.Procs; i++ {
		go a.consumer(ctx, producer, resultC)
	}
	return resultC
}

func (a *Analyzer) BytesRead() int64 {
	return atomic.LoadInt64(&a.bytesRead)
}

func (a *Analyzer) startProducer(ctx context.Context) <-chan LineMsg {
	logLines := make(chan LineMsg, bufSize)
	currentOffset := a.Offset
	reader := bufio.NewReaderSize(a.Log, 32*1024*1024)
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer close(logLines)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil && err != io.EOF {
				fatal("error while scanning log: %s: %s\n", a.Path, err)
			}
			atomic.AddInt64(&a.bytesRead, int64(len(line)))
			if len(line) > 0 {
				select {
				case <-ctx.Done():
					return
				case logLines <- LineMsg{Line: line, Offset: currentOffset}:
					currentOffset += int64(len(line))
				}
				if err == io.EOF {
					return
				}
			} else {
				return
			}
		}
	}()
	return logLines
}

func (a *Analyzer) consumer(ctx context.Context, producer <-chan LineMsg, results chan<- Result) {
	defer a.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-producer:
			if !ok {
				return
			}
			result := a.Func(msg.Line)
			if result != nil {
				result.Path = a.Path
				result.Offset = msg.Offset
				if !a.VerboseResults {
					result.Match = ""
				}
				select {
				case results <- *result:
				case <-ctx.Done():
				}
			}
		}
	}
}

/*
func NoopAnalyzerFunc(line []byte) *Result {
	return nil
}
*/
