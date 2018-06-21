// Records redirects encountered while making get requests for a list of domains
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
)

// domain -> redirect 1 -> redirect 2
// domain -> error:

// domain, 

type Result struct {
	Domain string
	RedirectChain []string
	Error error
}

func worker(input <-chan string, output chan<- *Result, done chan<- bool) {
	/*
	client := &http.Client(
		CheckRedirect: func() {
			//TODO:
		}
	) */

	fmt.Println("worker starting")
	for url := range input {
		output <- &Result{Domain:url}
	}

	fmt.Println("worker done")
	done <- true
}

func output(results <-chan *Result, done chan<- bool) {
	for result := range results {
		fmt.Printf("results: %s\n", result.Domain)
	}
	done <- true
}

func main() {
	numWorkers := flag.Int("workers", 500, "number of concurrent requests to allow")
	maxRedirects := flag.Int("max-redirects", 10, "maximum number of redirects to follow")
	// Use HEAD or GET?

	flag.Parse()

	fmt.Printf("%d %d\n", *numWorkers, *maxRedirects)

	workChan := make(chan string)
	outputChan := make(chan *Result)
	workerDoneChan := make(chan bool)
	outputDoneChan := make(chan bool)

	// Launch workers
	go output(outputChan, outputDoneChan)
	for i := 0; i < *numWorkers; i++ {
		go worker(workChan, outputChan, workerDoneChan)
	}

	// Read list of urls to process
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		nextUrl := scanner.Text()
		// Skip empty lines
		if nextUrl == "" {
			continue
		}
		if _, err := url.Parse(nextUrl); err != nil {
			panic(err)
		}
		workChan <- nextUrl
	}
	if scanner.Err() != nil {
		panic(scanner.Err())
	}

	// Notify workers that there's no more work
	close(workChan)

	// Wait for workers to finish
	for i := 0; i < *numWorkers; i++ {
		<-workerDoneChan
	}
	close(outputChan)
	<-outputDoneChan
}
