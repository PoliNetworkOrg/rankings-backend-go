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
	page, res, _, err := utils.LoadHttpDoc(constants.WebPolimiAvvisiFuturiStudentiUrl)
	if err != nil {
		log.Fatalf("Error while loading avvisi page. url %s. err: %v", constants.WebPolimiAvvisiFuturiStudentiUrl, err)
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
			page, _, _, err := utils.LoadHttpDoc(link)
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
	Id    string
	Url   *url.URL
	Pages []HtmlPage
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
	id := splitted[1]

	slog.Debug("start recursive download", "link", startingLink)
	htmlRanking := HtmlRanking{Url: url, Id: id, Pages: make([]HtmlPage, 0)}
	page, res, mainHtml, err := utils.LoadHttpDoc(startingLink)
	if err != nil {
		slog.Error("Could not load ranking main page.", "url", startingLink, "error", err)
		return htmlRanking
	}

	htmlRanking.Pages = append(htmlRanking.Pages, HtmlPage{ Id: id, Content: mainHtml })

	indexesHrefs := make([]string, 0)
	page.Find(".titolo a").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		indexesHrefs = append(indexesHrefs, href)
	})

	for _, href := range indexesHrefs {
		link := utils.PatchRelativeHref(href, res.Request.URL)
		page, indexRes, indexHtml, err := utils.LoadHttpDoc(link)
		if err != nil {
			slog.Error("Error while loading ranking sub-index page.", "url", link, "error", err)
			continue
		}
		htmlRanking.Pages = append(htmlRanking.Pages, HtmlPage{ Id: href, Content: indexHtml })

		ws := sync.WaitGroup{}
		page.Find(".TableDati td a").Each(func(_ int, e *goquery.Selection) {
			ws.Add(1)
			go func() {
				defer ws.Done()
				href, _ := e.Attr("href")
				link := utils.PatchRelativeHref(href, indexRes.Request.URL)
				_, _, tableHtml, err := utils.LoadHttpDoc(link)
				if err != nil {
					slog.Error("Could not load ranking table page. url %s. err: %v", link, err)
					return
				}

				htmlRanking.Pages = append(htmlRanking.Pages, HtmlPage{ Id: href, Content: tableHtml })
			}()
		})
		ws.Wait()
	}

	return htmlRanking
}
