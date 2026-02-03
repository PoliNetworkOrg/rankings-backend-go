package scraper

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/writer"
)

const (
	hexCombos = 1 << 16
)

// format: yyyy_20xxx_hhhh_html
// yyyy is ranking's year
// x is a number, h is an hex char, so all the combinations would be 47'775'744.
// It is probable by inspection, that those are static IDs representing the ranking phase
// so we can check only for those to save on the number of links to check.
// We extract programmatically IDs from saved html folders name, which represent the phase IDs
func extractPhaseIDs(savedHtmlsAbsPath string) []uint {
	out := []uint{}
	// get all saved HTML folders
	entries, err := utils.GetEntriesInFolder(savedHtmlsAbsPath)
	if err != nil {
		slog.Error("[bruteforce] Could not read saved HTML entries from local folder", "folderAbsPath", savedHtmlsAbsPath)
	}

	for _, entry := range entries {
		name := entry.Name() // e.g. 2020_20002_html; 2024_20103_2d5d_html
		parts := strings.Split(name, "_")
		if len(parts) < 3 {
			slog.Warn("[bruteforce] Saved HTML folder name is not recognized", "name", name)
			continue
		}

		rawID := parts[1] // e.g. 20002; 20103
		parsedID, err := strconv.Atoi(strings.TrimPrefix(rawID, "2"))
		if err != nil {
			slog.Error("[bruteforce] Found HTML phase ID not integer", "rawID", parts[1], "folder_name", name)
		}

		uintID := uint(parsedID) // e.g. 2; 103
		alreadyKnown := slices.Contains(out, uintID)

		// slog.Debug("[bruteforce] extractPhaseIDs single", "ID", uintID, "isNew", !alreadyKnown)
		if !alreadyKnown {
			out = append(out, uintID)
		}
	}

	slices.SortStableFunc(out, func(i, j uint) int {
		return int(i) - int(j)
	})

	slog.Info("[bruteforce] final known phaseIDs", "data", out)

	return out
}

type Bruteforcer struct {
	Year       uint
	validLinks []string
	phaseIDs   []uint

	writer writer.Writer[[]string]
}

func NewBruteforcer(absOutDir, absSavedHtmlsDir string, year uint) *Bruteforcer {
	writer := writer.NewWriter[[]string](absOutDir)

	return &Bruteforcer{
		validLinks: []string{},
		Year:       year,
		writer:     writer,
		phaseIDs:   extractPhaseIDs(absSavedHtmlsDir),
	}
}

func (bf *Bruteforcer) getFilename() string {
	return fmt.Sprintf("valid_links_%d.json", bf.Year)
}

func (bf *Bruteforcer) write() {
	err := bf.writer.JsonWrite(bf.getFilename(), bf.validLinks, true)
	if err != nil {
		slog.Error("[bruteforce] error while writing working links to filesystem, printing to console links as fallback", "count", len(bf.validLinks), "path", bf.writer.GetFilePath(bf.getFilename()), "err", err)
		for link := range bf.validLinks {
			slog.Info("[bruteforce] FALLBACK working link found", "link", link)
		}
		return
	}

	slog.Info("[bruteforce] successfully written to file", "count", len(bf.validLinks), "year", bf.Year)
}

func (bf *Bruteforcer) ReadSavedValidLinks() []string {
	res, err := bf.writer.JsonRead(bf.getFilename())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Info("[bruteforce] saved valid links (for the specified year) file does not exist", "year", bf.Year, "path", bf.writer.GetFilePath(bf.getFilename()))
		} else {
			slog.Error("[bruteforce] error while reading already saved valid links file", "year", bf.Year, "path", bf.writer.GetFilePath(bf.getFilename()), "err", err)
		}
	}

	return res
}

func (bf *Bruteforcer) generateLink(phaseID uint, randomHex int) string {
	return fmt.Sprintf("https://%s/%d_2%04d_%04x_html/%d_2%04d_generale.html",
		constants.WebPolimiRisultatiAmmissioneDomainName,
		bf.Year,
		phaseID,
		randomHex,
		bf.Year,
		phaseID,
	)
}

func (bf *Bruteforcer) Start() []string {
	saved := bf.ReadSavedValidLinks()

	if len(saved) > 0 {
		slog.Info("[bruteforce] results for the specified year already exists, returning saved links", "year", bf.Year, "path", bf.writer.GetFilePath(bf.getFilename()))
		return saved
	}

	// total links to test
	combos := len(bf.phaseIDs) * hexCombos
	links := make([]string, combos)

	slog.Info("[bruteforce] started bruteforce, it might take a while", "year", bf.Year, "combos", combos)

	for i, id := range bf.phaseIDs {
		offset := i * hexCombos

		for randomHex := range hexCombos {
			idx := offset + randomHex
			links[idx] = bf.generateLink(id, randomHex)
		}
	}

	slog.Debug("[bruteforce] generated links to test", "count", combos, "first", links[0], "last", links[combos-1])
	results := utils.HttpHeadAll(links, 200, 1000, 10*time.Second)

	for _, result := range results {
		if result.StatusCode == 200 {
			slog.Info("[HTTP_HEAD] link 200", "link", result.Link, "statusCode", result.StatusCode)
			bf.validLinks = append(bf.validLinks, result.Link)
		}
	}

	slog.Info("[bruteforce] ended bruteforce", "year", bf.Year)
	bf.write()
	return bf.validLinks
}

// I leave here two knownPhaseIDs slices as backup
// the second is programatically generated with the function `extractPhaseIDs` after 2025 rankings
// [2 5 6 7 8 9 10 11 12 13 15 16 17 18 19 20 21 22 24 25 26 32 33 34 35 37 39 40 41 42 43 44 45 46 47 48 52 54 55 56 57 58 59 60 64 69 82 83 87 91 93 94 95 96 102 103 104]
// [2 5 6 7 8 9 10 11 12 13 17 19 20 21 22 24 25 26 39 40 41 42 43 44 45 47 48 54 55 56 57 58 59 60 64 69 82 83 87 91 93 94 95 96 102 103 104]
