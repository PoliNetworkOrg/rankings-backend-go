package scraper

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
	"github.com/PuerkitoBio/goquery"
)

func isRankingsNews(str string) bool {
	newsTesters := []string{
		"graduatorie", "graduatoria", "punteggi", "tol",
		"immatricolazioni", "immatricolazione", "punteggio",
		"matricola", "nuovi studenti",
	}

	for _, tester := range newsTesters {
		if strings.Contains(str, tester) {
			return true
		}
	}

	return false
}

type Manifesto struct {
	Name       string `json:"name"`
	Url        string `json:"url"`
	Location   string `json:"location"`
	DegreeType string `json:"type"`
}

func ScrapeManifesti(alreadyScraped []Manifesto) []Manifesto {
	urls := []string{constants.WebPolimiDesignUrl, constants.WebPolimiArchUrbUrl, constants.WebPolimiIngCivUrl, constants.WebPolimiIngInfIndUrl}
	// hrefs := []string{}
	out := alreadyScraped

	wg := sync.WaitGroup{}

	alreadyScrapedUrl := make([]string, len(alreadyScraped))
	for i, as := range alreadyScraped {
		alreadyScrapedUrl[i] = as.Url
	}

	for _, url := range urls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doc, res, err := loadDoc(url)
			if err != nil {
				log.Fatalf("Error while loading school url %s. err: %v", url, err)
			}

			var manHref string
			doc.Find(".frame a").Each(func(i int, e *goquery.Selection) {
				text := strings.ToLower(e.Text())
				href, ok := e.Attr("href")
				if strings.Contains(text, "piano di studi") && ok {
					manHref = href
				}
			})

			doc, res, err = loadDoc(manHref)
			if err != nil {
				log.Fatalf("Error while loading manifest url %s. err: %v", manHref, err)
			}

			finalUrl := res.Request.URL
			doc.Find("#id_combocds > tbody > tr:nth-child(3) > td.ElementInfoCard2.left > select > optgroup").Each(func(i int, group *goquery.Selection) {
				degreeType, ok := group.Attr("label")
				if !ok {
					return
				}

				degreeType = strings.Split(degreeType, " -")[0]

				group.Children().Each(func(i int, opt *goquery.Selection) {
					courseName := opt.Text()
					courseName = strings.Split(courseName, " (")[0]

					value, err := strconv.ParseUint(opt.AttrOr("value", "0"), 10, 64)
					if err != nil {
						log.Fatal(err)
					}

					optUrl := *finalUrl
					q := optUrl.Query()
					q.Set("k_corso_la", strconv.FormatUint(value, 10))
					q.Del("__pj1")
					q.Del("__pj0")
					optUrl.RawQuery = q.Encode()

					if slices.Contains(alreadyScrapedUrl, optUrl.String()) {
						slog.Debug("url already scraped, skipping...", "url", optUrl.String())
						return
					}

					slog.Debug("found new manifesti url, scraping...", "url", optUrl.String())
					mandoc, _, err := loadDoc(optUrl.String())
					if err != nil {
						log.Fatal(err)
					}

					mandoc.Find("td.CenterBar table.BoxInfoCard tr:nth-child(4) td:nth-child(4)").First().Each(func(i int, loc *goquery.Selection) {
						locations := strings.Split(loc.Text(), ",")
						for _, location := range locations {
							newMan := Manifesto{
								Name:       strings.TrimSpace(courseName),
								Url:        optUrl.String(),
								Location:   strings.TrimSpace(location),
								DegreeType: strings.TrimSpace(degreeType),
							}
							out = append(out, newMan)
						}
					})
				})
			})
		}()
	}

	wg.Wait()

	// because of there are some courses shared between schools, they appears twice
	// in the list, while we want them only once.
	// In the future we could also track the School, so it would not cause the issue.
	// e.g. Design & Engineering (Des, 3I), Geoinformatics Engineering (3I, IngCiv)
	cleanOut := make([]Manifesto, 0, len(out))
	for i, m1 := range out[:len(out)] {
		count := 0
		for _, m2 := range out[i+1:] {
			if reflect.DeepEqual(m1, m2) {
				count++
			}
		}

		if count == 0 {
			cleanOut = append(cleanOut, m1)
		} else {
			// found a duplicate, not adding. 
			// it will be added when m1 -> m2 -> ... -> mn, with mn last duplicate
			slog.Debug("scraper manifesti: found duplicate", "manifesto", m1)
		}
	}

	return cleanOut
}

func loadDoc(url string) (*goquery.Document, *http.Response, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, nil, err
	}

	return doc, res, nil
}
