package utils

import (
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

func LoadHttpDoc(url string) (*goquery.Document, *http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, nil, fmt.Errorf("HTTP code is not 200. Status: %s", res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, nil, err
	}

	return doc, res, nil
}
