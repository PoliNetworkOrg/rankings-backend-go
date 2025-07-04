package utils

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func LoadHttpHtml(url string) (*goquery.Document, *http.Response, []byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
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
