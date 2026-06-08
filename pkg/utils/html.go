package utils

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Global client sharing cookie state
var scraperClient *http.Client

func init() {
	jar, _ := cookiejar.New(nil)
	scraperClient = &http.Client{
		// Polimi backend queries can take up to 30-40 seconds under load
		Timeout: 45 * time.Second,
		Jar:     jar,
	}
}

func LoadHttpHtml(url string) (*goquery.Document, *http.Response, []byte, error) {
	slog.Debug("Requested HTML from HTTP", "url", url)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "it-IT,it;q=0.9,en-US;q=0.8,en;q=0.7")
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, nil, err
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, nil, nil, fmt.Errorf("HTTP code is not 200. Status: %s", res.Status)
	}

	htmlBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBytes))
	if err != nil {
		return nil, nil, nil, err
	}

	slog.Debug("Recieved HTML from HTTP Request", "url", url, "length", len(htmlBytes))
	return doc, res, htmlBytes, nil
}

func LoadLocalHtml(data []byte) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func PatchRelativeHref(href string, url *url.URL) string {
	parsed, err := url.Parse(href)
	if err != nil {
		slog.Error("Could not patch relative href, error while parsing it as url.Url", "href", href, "error", err)
		return href
	}

	if !strings.HasPrefix(url.Path, "/") {
		splitted := strings.Split(url.Path, "/")
		splitted[len(splitted)-1] = href
		parsed.Path = strings.Join(splitted, "/")
	}

	if len(parsed.Scheme) == 0 {
		parsed.Scheme = url.Scheme
	}

	if len(parsed.Host) == 0 {
		parsed.Host = url.Host
	}

	return parsed.String()
}

func GetFirstTextFragment(s *goquery.Selection) (string, error) {
	innerHtml, err := s.Html()
	if err != nil {
		return "", err
	}
	splittedHtml := strings.Split(innerHtml, "<br/>") // they love <br/> to separate languages
	unescaped := html.UnescapeString(splittedHtml[0])
	return unescaped, nil
}
