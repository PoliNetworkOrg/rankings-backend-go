package scraper

import (
	"context"
	"fmt"
	"log/slog"
	neturl "net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/sync/errgroup"
)

type Manifesto struct {
	Name       string `json:"name"`
	Url        string `json:"url"`
	Location   string `json:"location"`
	DegreeType string `json:"type"`
}

func ScrapeManifesti(alreadyScraped []Manifesto) []Manifesto {
	schoolURLs := []string{constants.WebPolimiDesignUrl, constants.WebPolimiArchUrbUrl, constants.WebPolimiIngCivUrl, constants.WebPolimiIngInfIndUrl}
	out := alreadyScraped
	mu := sync.Mutex{}

	eg, ctx := errgroup.WithContext(context.Background())
	eg.SetLimit(1)

	alreadyScrapedSet := make(map[string]struct{}, len(alreadyScraped))
	for _, as := range alreadyScraped {
		alreadyScrapedSet[as.Url] = struct{}{}
	}

	for _, url := range schoolURLs {
		eg.Go(func() error {
			scraper := NewScraper(2, 1)

			ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			slog.Debug("fetching school's website to get manifests page url", "url", url)
			doc, _, _, err := scraper.LoadHttpHtmlRetry(ctx, url, 5)
			if err != nil {
				return fmt.Errorf("error while loading school url %s. err: %w", url, err)
			}

			var manHref string
			var innerError error
			doc.Find(".frame a").Each(func(i int, e *goquery.Selection) {
				if innerError != nil {
					return
				}

				text := strings.ToLower(e.Text())
				href, ok := e.Attr("href")
				if strings.Contains(text, "piano di studi") && ok {
					if hrefURL, err := neturl.Parse(href); err == nil {
						manHref = hrefURL.String()
					}
				}
			})

			if innerError != nil {
				return innerError
			}

			if manHref == "" {
				return fmt.Errorf("manifest link not found for %s", url)
			}

			doc, resURL, _, err := scraper.LoadHttpHtmlRetry(ctx, manHref, 5)
			slog.Debug("fetching school's manifest list", "url", manHref)
			if err != nil {
				return fmt.Errorf("error while loading manifest url %s: %w", manHref, err)
			}

			doc.Find("#id_combocds > tbody > tr:nth-child(3) > td.ElementInfoCard2.left > select > optgroup").Each(func(i int, group *goquery.Selection) {
				degreeType, ok := group.Attr("label")
				if !ok {
					return
				}

				degreeType = strings.Split(degreeType, " -")[0]

				group.Children().Each(func(i int, opt *goquery.Selection) {
					if innerError != nil {
						return
					}

					courseName := opt.Text()
					courseName = strings.Split(courseName, " (")[0]

					value, err := strconv.ParseUint(opt.AttrOr("value", "0"), 10, 64)
					if err != nil {
						innerError = err
						return
					}

					optURL := *resURL
					q := optURL.Query()
					q.Set("k_corso_la", strconv.FormatUint(value, 10))
					q.Del("__pj1")
					q.Del("__pj0")
					optURL.RawQuery = q.Encode()

					if _, ok := alreadyScrapedSet[optURL.String()]; ok {
						slog.Debug("url already scraped, skipping...", "url", optURL.String())
						return
					}

					slog.Info("found new manifesti url, scraping...", "url", optURL.String())
					mandoc, _, _, err := scraper.LoadHttpHtmlRetry(ctx, optURL.String(), 5)
					if err != nil {
						innerError = err
						return
					}

					mandoc.Find("td.CenterBar table.BoxInfoCard tr:nth-child(4) td:nth-child(4)").First().Each(func(i int, loc *goquery.Selection) {
						locations := strings.Split(loc.Text(), ",")
						for _, location := range locations {
							newMan := Manifesto{
								Name:       strings.TrimSpace(courseName),
								Url:        optURL.String(),
								Location:   strings.TrimSpace(location),
								DegreeType: strings.TrimSpace(degreeType),
							}

							mu.Lock()
							out = append(out, newMan)
							mu.Unlock()
						}
					})
				})
			})

			return innerError
		})
	}

	err := eg.Wait()
	if err != nil {
		slog.Error("There was an error while scraping manifesti", "error", err)
	}

	// because of there are some courses shared between schools, they appears twice
	// in the list, while we want them only once.
	// In the future we could also track the School, so it would not cause the issue.
	// e.g. Design & Engineering (Des, 3I), Geoinformatics Engineering (3I, IngCiv)
	cleanOut := make([]Manifesto, 0, len(out))
	seen := make(map[Manifesto]struct{}, len(out))

	for _, m := range out {
		if _, ok := seen[m]; ok {
			continue
		}

		seen[m] = struct{}{}
		cleanOut = append(cleanOut, m)
	}

	return cleanOut
}
