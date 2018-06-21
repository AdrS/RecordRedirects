// Records redirects encountered while making get requests for a list of domains
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"net/url"
	"strings"
	"time"
	"os"
)

type Result struct {
	URL string             `json:"url"`
	RedirectChain []string `json:"chain,omitempty"`
	Error string           `json:"error,omitempty"`
}

func worker(input <-chan string, output chan<- *Result, done chan<- bool) {
	for url := range input {
		// Use HTTP as default scheme
		if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
			url = "http://" + url
		}
		result := &Result{URL:url}
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(result.RedirectChain) >= maxRedirects {
					return errors.New("too many redirects")
				}
				result.RedirectChain = append(result.RedirectChain, req.URL.String())
				return nil
			},
			Timeout: 30*time.Second,
		}
		_, err := client.Get(url)
		if err != nil {
			result.Error = err.Error()
		}
		output <- result
	}
	done <- true
}

func output(results <-chan *Result, done chan<- bool) {
	for result := range results {
		m, err := json.Marshal(result)
		if err != nil {
			panic(err)
		}
		m = append(m, byte('\n'))
		n, err := os.Stdout.Write(m)
		if err != nil || n != len(m) {
			panic(err)
		}
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
