package scraper

import (
	"errors"
	"log/slog"
	"os"
	"slices"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/writer"
)

type LinksManager struct {
	alreadyScrapedLinks []string
	alreadyBrokenLinks  []string
	newScrapedLinks     []string
	newBrokenLinks      []string

	writer writer.Writer[[]string]
}

func NewLinksManager(absOutDir string) *LinksManager {
	writer := writer.NewWriter[[]string](absOutDir)
	lm := &LinksManager{writer: writer}
	lm.readAlreadyScraped()
	lm.readAlreadyBroken()

	return lm
}

func (lm *LinksManager) PrintState(context string) {
	slog.Info("[links-manager] STATE", "context", context, "alreadyScraped", len(lm.alreadyScrapedLinks), "alreadyBroken", len(lm.alreadyBrokenLinks), "newScraped", len(lm.newScrapedLinks), "newBroken", len(lm.newBrokenLinks))
}

func (lm *LinksManager) readAlreadyBroken() {
	path := lm.writer.GetFilePath(constants.OutputBrokenLinksFilename)
	links, err := lm.writer.JsonRead(constants.OutputBrokenLinksFilename)
	if err == nil {
		lm.alreadyBrokenLinks = links
		slog.Info("[links-manager] successfully parsed already broken links file", "count", len(links), "path", path)
		return
	}

	if errors.Is(err, os.ErrNotExist) {
		slog.Warn("[links-manager] already broken links file does not exist, creating empty...", "path", path)
		if err := lm.writer.JsonWrite(constants.OutputBrokenLinksFilename, []string{}, true); err != nil {
			slog.Error("[links-manager] cannot create empty broken links file", "path", path, "err", err)
		}
		return
	}

	slog.Error("[links-manager] error while reading already broken links file", "err", err)
}

func (lm *LinksManager) readAlreadyScraped() {
	path := lm.writer.GetFilePath(constants.OutputScrapedLinksFilename)
	links, err := lm.writer.JsonRead(constants.OutputScrapedLinksFilename)
	if err == nil {
		lm.alreadyScrapedLinks = links
		slog.Info("[links-manager] successfully parsed already scraped links file", "count", len(links), "path", path)
		return
	}

	if errors.Is(err, os.ErrNotExist) {
		slog.Warn("[links-manager] already scraped links file does not exist, creating empty...", "path", path)
		if err := lm.writer.JsonWrite(constants.OutputScrapedLinksFilename, []string{}, true); err != nil {
			slog.Error("[links-manager] cannot create empty scraped links file", "path", path, "err", err)
		}
		return
	} else {
		slog.Error("[links-manager] error while reading already scraped links file", "err", err)
	}
}

func (lm *LinksManager) mergeLinks() ([]string, []string) {
	mergedScraped := slices.Concat(lm.alreadyScrapedLinks, lm.newScrapedLinks)
	mergedBroken := slices.Concat(lm.alreadyBrokenLinks, lm.newBrokenLinks)
	slices.Sort(mergedScraped)
	slices.Sort(mergedBroken)
	return mergedScraped, mergedBroken
}

func (lm *LinksManager) writeScraped(mergedScraped []string, force bool) {
	path := lm.writer.GetFilePath(constants.OutputScrapedLinksFilename)
	if err := lm.writer.JsonWrite(constants.OutputScrapedLinksFilename, mergedScraped, true); err != nil {
		slog.Error("[links-manager] cannot write scraped links to filesystem, printing to console new links as fallback", "newLinksCount", len(lm.newScrapedLinks), "path", path, "err", err)
		for link := range lm.newScrapedLinks {
			slog.Info("[links-manager] FALLBACK new link scraped", "link", link)
		}
		return
	}

	slog.Info("[links-manager] successfully written to file", "totalCount", len(mergedScraped), "newLinksCount", len(lm.newScrapedLinks), "alreadyScrapedCount", len(lm.alreadyScrapedLinks), "force", force)
}

func (lm *LinksManager) writeBroken(mergedBroken []string, force bool) {
	path := lm.writer.GetFilePath(constants.OutputBrokenLinksFilename)
	if err := lm.writer.JsonWrite(constants.OutputBrokenLinksFilename, mergedBroken, true); err != nil {
		slog.Error("[links-manager] cannot write broken links to filesystem, printing to console new links as fallback", "newLinksCount", len(lm.newBrokenLinks), "path", path, "err", err)
		for link := range lm.newBrokenLinks {
			slog.Info("[links-manager] FALLBACK new link broken", "link", link)
		}
		return
	}

	slog.Info("[links-manager] successfully written to file", "totalCount", len(mergedBroken), "newLinksCount", len(lm.newBrokenLinks), "alreadyBrokenCount", len(lm.alreadyBrokenLinks), "force", force)
}

func (lm *LinksManager) Write(force bool) {
	if len(lm.newBrokenLinks) == 0 && len(lm.newScrapedLinks) == 0 && !force {
		slog.Info("[links-manager] no new links, nothing to write")
		return
	}

	mergedScraped, mergedBroken := lm.mergeLinks()
	slog.Info("Writing links, counts", "alreadyScraped", len(lm.alreadyScrapedLinks), "alreadyBroken", len(lm.alreadyBrokenLinks), "newScraped", len(lm.newScrapedLinks), "newBroken", len(lm.newBrokenLinks), "mergedScraped", len(mergedScraped), "mergedBroken", len(mergedBroken))

	if len(lm.newScrapedLinks) > 0 || force {
		lm.writeScraped(mergedScraped, force)
	}

	if len(lm.newBrokenLinks) > 0 || force {
		lm.writeBroken(mergedBroken, force)
	}
}

func (lm *LinksManager) FilterNewLinks(links []string) []string {
	filtered := []string{}
	for _, link := range links {
		if !slices.Contains(lm.alreadyScrapedLinks, link) && !slices.Contains(lm.newBrokenLinks, link) {
			filtered = append(filtered, link)
		}
	}

	return filtered
}

func (lm *LinksManager) SetNewLinks(scraped, broken []string) {
	lm.newScrapedLinks = lm.FilterNewLinks(scraped)
	lm.newBrokenLinks = lm.FilterNewLinks(broken)
}
