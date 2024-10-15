package scraper

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
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
	Name     string `json:"name"`
	Url      string `json:"url"`
	Location string `json:"location"`
}

func ScrapeManifesti() []Manifesto {
	urls := []string{constants.WebPolimiDesignUrl, constants.WebPolimiArchUrbUrl, constants.WebPolimiIngCivUrl, constants.WebPolimiIngInfIndUrl}
	// hrefs := []string{}
	out := []Manifesto{}

	wg := sync.WaitGroup {}

	for _, url := range urls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doc, res, err := loadDoc(url)
			if err != nil {
				log.Fatalf("WHAT THE FUCK???? \nerr: %v", err)
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
				log.Fatalf("WHAT THE FUCK???? \nerr: %v", err)
			}

			finalUrl := res.Request.URL
			doc.Find("#id_combocds > tbody > tr:nth-child(3) > td.ElementInfoCard2.left > select > optgroup").Each(func(i int, group *goquery.Selection) {
				label, ok := group.Attr("label")
				if !ok {
					return
				}

				label = strings.Split(label, " -")[0]

				group.Children().Each(func(i int, opt *goquery.Selection) {
					text := opt.Text()
					text = strings.Split(text, " (")[0]

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

					slog.Info("optgroup", "label", label, "opt", text, "value", value, "link", optUrl.String())
					mandoc, _, err := loadDoc(optUrl.String())
					if err != nil {
						log.Fatal(err)
					}

					mandoc.Find("td.CenterBar table.BoxInfoCard tr:nth-child(4) td:nth-child(4)").Each(func(i int, loc *goquery.Selection) {
						locations := strings.Split(loc.Text(), ",")
						for _, location := range locations {
							newMan := Manifesto{
								Name:     strings.TrimSpace(text),
								Url:      optUrl.String(),
								Location: strings.TrimSpace(location),
							}
							out = append(out, newMan)
						}
					})
				})
			})
		}()
	}

	wg.Wait()

	return out
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
