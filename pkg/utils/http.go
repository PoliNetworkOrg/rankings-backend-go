package utils

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type HeadResult struct {
	Link       string `json:"link"`
	StatusCode int    `json:"statusCode"`
	Err        error
}

func HttpHeadAll(
	links []string,
	maxWorkers int, // number of concurrent HTTP requests
	rps int, // requests per second (0 = unlimited)
	reqTimeout time.Duration, // per-request timeout
) []HeadResult {
	n := len(links)
	results := make([]HeadResult, n)

	// Tuned transport to reuse connections efficiently.
	tr := &http.Transport{
		MaxIdleConns:        maxWorkers * 2,
		MaxIdleConnsPerHost: maxWorkers,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	client := &http.Client{
		Transport: tr,
		// Do not set Client.Timeout so we can use per-request contexts.
	}

	// Job channel carries indexes into links/results
	jobs := make(chan int)
	var wg sync.WaitGroup

	// Optional rate limiter ticker
	var tickCh <-chan time.Time
	var ticker *time.Ticker
	if rps > 0 {
		interval := time.Second / time.Duration(rps)
		if interval <= 0 {
			interval = time.Millisecond // fallback
		}
		ticker = time.NewTicker(interval)
		tickCh = ticker.C
	}

	// Start worker goroutines
	for w := range maxWorkers {
		go func(workerID int) {
			for idx := range jobs {
				// rate limit if requested
				if tickCh != nil {
					<-tickCh
				}

				link := links[idx]
				result := HeadResult{Link: link, Err: nil}

				ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
				req, err := http.NewRequestWithContext(ctx, "HEAD", link, nil)
				if err != nil {
					result.Err = err
					result.StatusCode = 500
					results[idx] = result
					slog.Error("[HTTP_HEAD] link error", "idx", idx, "link", link)
					cancel()
					wg.Done()
					continue
				}

				resp, err := client.Do(req)
				if err != nil {
					result.Err = err
					result.StatusCode = 500
					results[idx] = result
					slog.Error("[HTTP_HEAD] link error", "idx", idx, "link", link)
					cancel()
					wg.Done()
					continue
				}

				result.StatusCode = resp.StatusCode
				results[idx] = result
				if resp.StatusCode == 200 {
					slog.Debug("[HTTP_HEAD] link 200", "idx", idx, "link", link, "statusCode", resp.StatusCode)
				} else {
					slog.Info("[HTTP_HEAD] link not 200", "idx", idx, "link", link, "statusCode", resp.StatusCode)
				}
				cancel()
				wg.Done()
			}
		}(w)
	}

	// Enqueue jobs
	wg.Add(n)
	for i := range n {
		jobs <- i
	}
	close(jobs)

	// Wait for all jobs to finish
	wg.Wait()
	if ticker != nil {
		ticker.Stop()
	}
	return results
}
