package main

import (
	"bufio"
	"context"
	"io"
	"sync"
)

const bufSize = 1000

type Analyzer struct {
	Procs int
	Log   io.Reader
	Func  AnalyzerFunc
}

type AnalyzerFunc func([]byte) *Result

type Result struct {
	Match string
	Err   error
}

func (a *Analyzer) Go(ctx context.Context) <-chan Result {
	resultC := make(chan Result)
	var wg sync.WaitGroup
	wg.Add(a.Procs)
	go func() {
		wg.Wait()
		close(resultC)
	}()
	producer := a.startProducer(ctx)
	for i := 0; i < a.Procs; i++ {
		go a.consumer(ctx, producer, resultC, &wg)
	}
	return resultC
}

func (a *Analyzer) startProducer(ctx context.Context) <-chan []byte {
	result := make(chan []byte, bufSize)
	reader := bufio.NewReaderSize(a.Log, 32*1024*1024)
	scanner := bufio.NewScanner(reader)
	// 1 MB scanner buffer
	// buf := make([]byte, 1024*1024)
	// scanner.Buffer(buf, len(buf))
	go func() {
		for scanner.Scan() {
			line := scanner.Bytes()
			select {
			case <-ctx.Done():
				close(result)
				return
			case result <- line:
			}
		}
		if err := scanner.Err(); err != nil {
			fatal("error while scanning log: %s", err)
		}
		close(result)
	}()
	return result
}

func (a *Analyzer) consumer(ctx context.Context, producer <-chan []byte, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-producer:
			if !ok {
				return
			}
			result := a.Func(line)
			if result != nil {
				select {
				case results <- *result:
				case <-ctx.Done():
				}
			}
		}
	}
}

func NoopAnalyzerFunc(line []byte) *Result {
	return nil
}
