// Records redirects encountered while making get requests for a list of domains
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
)

// Number of workers

// domain -> redirect 1 -> redirect 2
// domain -> error:

func worker(input <-chan string, ouput chan<- string, done chan<- bool) {
	/*
	client := &http.Client(
		CheckRedirect: func() {
			//TODO:
		}
	} */

	fmt.Println("worker starting")
	for url := <-input; url != ""; url = <-input {
		fmt.Printf("working on %s\n", url)
	}

	fmt.Println("worker done")
	done <- true
}

func main() {
	numWorkers := flag.Int("workers", 500, "number of concurrent requests to allow")
	maxRedirects := flag.Int("max-redirects", 10, "maximum number of redirects to follow")
	flag.Parse()

	fmt.Printf("%d %d\n", *numWorkers, *maxRedirects)

	workChan := make(chan string)
	outputChan := make(chan string)
	doneChan := make(chan bool)

	// Launch workers
	for i := 0; i < *numWorkers; i++ {
		go worker(workChan, outputChan, doneChan)
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
	for i := 0; i < *numWorkers; i++ {
		workChan <- ""
	}

	// Wait for workers to finish
	for i := 0; i < *numWorkers; i++ {
		<-doneChan
	}
}
