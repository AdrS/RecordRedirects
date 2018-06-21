// Records redirects encountered while making get requests for a list of domains
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

type Result struct {
	Domain string
	RedirectChain []string
	Error error
}

func worker(input <-chan string, output chan<- *Result, done chan<- bool) {
	for domain := range input {
		result := &Result{Domain:domain}
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				fmt.Printf("%d vs %d\n", len(result.RedirectChain), maxRedirects)
				if len(result.RedirectChain) >= maxRedirects {
					return errors.New("too many redirects")
				}
				result.RedirectChain = append(result.RedirectChain, req.URL.String())
				return nil
			},
		}
		_, err := client.Get("http://" + domain)
		result.Error = err
		output <- result
	}
	done <- true
}

func output(results <-chan *Result, done chan<- bool) {
	for result := range results {
		//fmt.Printf("results: %s\n", result.Domain)
		fmt.Println(result)
	}
	done <- true
}

var maxRedirects int

func main() {
	var numWorkers int
	flag.IntVar(&numWorkers, "workers", 500, "number of concurrent requests to allow")
	flag.IntVar(&maxRedirects, "max-redirects", 10, "maximum number of redirects to follow")
	// Use HEAD or GET?

	flag.Parse()

	fmt.Printf("%d %d\n", numWorkers, maxRedirects)

	workChan := make(chan string)
	outputChan := make(chan *Result)
	workerDoneChan := make(chan bool)
	outputDoneChan := make(chan bool)

	// Launch workers
	go output(outputChan, outputDoneChan)
	for i := 0; i < numWorkers; i++ {
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
	for i := 0; i < numWorkers; i++ {
		<-workerDoneChan
	}
	close(outputChan)
	<-outputDoneChan
}
