package scraper

import (
	"log"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
	"github.com/PuerkitoBio/goquery"
)

func ScrapeRankingsLinks(savedLinks []string) []string {
	links := scrapeAvvisiPage()
	slog.Debug("output of scrapeAvvisiPage", "count", len(links))
	newLinks := make([]string, 0)
	for _, link := range links {
		if !slices.Contains(savedLinks, link) {
			newLinks = append(newLinks, link)
		}
	}

	return newLinks
}

func scrapeAvvisiPage() []string {
	page, res, _, err := utils.LoadHttpHtml(constants.WebPolimiAvvisiFuturiStudentiUrl)
	if err != nil {
		log.Fatalf("Error while loading avvisi page. url %s. err: %w", constants.WebPolimiAvvisiFuturiStudentiUrl, err)
	}

	newsLinks := make([]string, 0)
	rankingsLinks := make([]string, 0)
	page.Find(".news .card a.btn").Each(func(_ int, e *goquery.Selection) {
		title, _ := e.Attr("title")
		href, _ := e.Attr("href")
		if isRankingsNews(title) {
			link := utils.PatchRelativeHref(href, res.Request.URL)
			newsLinks = append(newsLinks, link)
		}
	})

	wg := sync.WaitGroup{}

	for _, link := range newsLinks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			page, _, _, err := utils.LoadHttpHtml(link)
			if err != nil {
				slog.Error("Error while loading a news page, skipping...", "url", link, "error", err)
			}

			page.Find(".news-text-wrap a").Each(func(_ int, e *goquery.Selection) {
				href, _ := e.Attr("href")
				url, err := url.Parse(href)
				if err != nil {
					slog.Error("Error while parsing the url of an article <a> tag", "url", link, "error", err)
					return
				}

				if url.Host == constants.WebPolimiRisultatiAmmissioneDomainName {
					link := utils.PatchRelativeHref(href, res.Request.URL)
					rankingsLinks = append(rankingsLinks, link)
				}
			})
		}()
	}

	wg.Wait()
	return rankingsLinks
}

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

type HtmlPage struct {
	Id      string
	Content []byte
}

type HtmlRanking struct {
	Id        string
	Url       *url.URL
	Index     HtmlPage
	ByMerit   []HtmlPage
	ById      []HtmlPage
	ByCourse  []HtmlPage
	PageCount int
}

func DownloadRankings(startingLinks []string) []HtmlRanking {
	ws := sync.WaitGroup{}
	out := make([]HtmlRanking, 0)
	for _, link := range startingLinks {
		ws.Add(1)
		go func() {
			defer ws.Done()
			htmlRanking := ScrapeRecursiveRankingHtmls(link)
			out = append(out, htmlRanking)
		}()
	}

	ws.Wait()
	return out
}

func ScrapeRecursiveRankingHtmls(startingLink string) HtmlRanking {
	url, _ := url.Parse(startingLink)
	splitted := strings.Split(url.Path, "/")
	count := 0
	id := splitted[1]

	slog.Debug("start recursive download", "link", startingLink)
	htmlRanking := HtmlRanking{Url: url, Id: id, PageCount: 0}
	page, res, mainHtml, err := utils.LoadHttpHtml(startingLink)
	if err != nil {
		slog.Error("Could not load ranking main page.", "url", startingLink, "error", err)
		return htmlRanking
	}

	htmlRanking.Index = HtmlPage{Id: id, Content: mainHtml}
	count++

	indexesHrefs := make([]string, 0)
	page.Find(".titolo a").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		indexesHrefs = append(indexesHrefs, href)
	})

	for _, href := range indexesHrefs {
		link := utils.PatchRelativeHref(href, res.Request.URL)
		page, indexRes, _, err := utils.LoadHttpHtml(link)
		if err != nil {
			slog.Error("Error while loading ranking sub-index page.", "url", link, "error", err)
			continue
		}

		pages := make([]HtmlPage, 0)

		ws := sync.WaitGroup{}
		page.Find(".TableDati td a").Each(func(_ int, e *goquery.Selection) {
			ws.Add(1)
			go func() {
				defer ws.Done()
				href, _ := e.Attr("href")
				link := utils.PatchRelativeHref(href, indexRes.Request.URL)
				_, _, tableHtml, err := utils.LoadHttpHtml(link)
				if err != nil {
					slog.Error("Could not load ranking table page. url %s. err: %w", link, err)
					return
				}

				pages = append(pages, HtmlPage{Id: href, Content: tableHtml})
			}()
		})
		ws.Wait()

		// IMPORTANT!!
		// ByCourse MUST BE THE FIRST IF STATEMENT
		// otherwise it will match ByMerit also for ByCourse Index
		if strings.HasSuffix(href, constants.HtmlRankingUrl_IndexSuffix_ByCourse) {
			slog.Debug("pattern matched index href with ByCourse", "href", href)
			htmlRanking.ByCourse = pages
		} else if strings.HasSuffix(href, constants.HtmlRankingUrl_IndexSuffix_ById) {
			slog.Debug("pattern matched index href with ById", "href", href)
			htmlRanking.ById = pages
		} else if strings.HasSuffix(href, constants.HtmlRankingUrl_IndexSuffix_ByMerit) {
			slog.Debug("pattern matched index href with ByMerit", "href", href)
			htmlRanking.ByMerit = pages
		} else {
			slog.Error("Index not recognized, please investigate.", "index_href", href, "index_url", link)
			continue
		}

		count += len(pages)
	}

	htmlRanking.PageCount = count
	htmlRanking.Debug()
	return htmlRanking
}

func (r *HtmlRanking) Debug() {
	slog.Debug("HtmlRanking", "id", r.Id, "byCourse", len(r.ByCourse), "byMerit", len(r.ByMerit), "byId", len(r.ById))
}
