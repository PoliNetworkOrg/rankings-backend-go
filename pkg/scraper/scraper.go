package scraper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/time/rate"
)

// Scraper handles concurrent, rate-limited scraping
type Scraper struct {
	limiter    *rate.Limiter
	httpClient *http.Client
}

func NewScraper(requestsPerSecond float64, burst int) *Scraper {
	jar, _ := cookiejar.New(nil)

	return &Scraper{
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
			Jar:     jar,
		},
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
	}
}

func (s *Scraper) LoadHttpHtml(
	ctx context.Context,
	targetURL string,
) (*goquery.Document, *url.URL, []byte, error) {
	return s.loadHTTPHTMLOnce(ctx, targetURL)
}

func (s *Scraper) LoadHttpHtmlRetry(
	ctx context.Context,
	targetURL string,
	maxAttempts int,
) (*goquery.Document, *url.URL, []byte, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		doc, finalURL, body, err := s.loadHTTPHTMLOnce(ctx, targetURL)
		if err == nil {
			return doc, finalURL, body, nil
		}

		lastErr = err

		if ctx.Err() != nil {
			return nil, nil, nil, ctx.Err()
		}

		if !isRetryableError(err) || attempt == maxAttempts {
			break
		}

		backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, nil, ctx.Err()
		case <-timer.C:
		}
	}

	return nil, nil, nil, fmt.Errorf(
		"failed after %d attempts for %s: %w",
		maxAttempts,
		targetURL,
		lastErr,
	)
}

func (s *Scraper) loadHTTPHTMLOnce(
	ctx context.Context,
	targetURL string,
) (*goquery.Document, *url.URL, []byte, error) {
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("rate limiter error: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		targetURL,
		nil,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	req.Header.Set(
		"User-Agent",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 "+
			"(KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36",
	)
	req.Header.Set(
		"Accept",
		"text/html,application/xhtml+xml,application/xml;q=0.9,"+
			"image/webp,*/*;q=0.8",
	)
	req.Header.Set("Accept-Language", "it-IT,it;q=0.9")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, nil, nil, &httpStatusError{
			StatusCode: res.StatusCode,
			Status:     res.Status,
			URL:        targetURL,
			Body:       string(body),
		}
	}

	htmlBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBytes))
	if err != nil {
		return nil, nil, nil, err
	}

	return doc, res.Request.URL, htmlBytes, nil
}

type httpStatusError struct {
	StatusCode int
	Status     string
	URL        string
	Body       string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf(
		"unexpected HTTP status %d (%s) for %s: %s",
		e.StatusCode,
		e.Status,
		e.URL,
		e.Body,
	)
}

func isRetryableError(err error) bool {
	var statusErr *httpStatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode == http.StatusTooManyRequests ||
			(statusErr.StatusCode >= 500 && statusErr.StatusCode <= 599)
	}

	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	return false
}
