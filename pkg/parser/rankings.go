package parser

import (
	"fmt"
	"log/slog"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/writer"
)

type CourseStatus struct {
	Title     string `json:"title"`
	Location  string `json:"location"`
	Position  uint16 `json:"position"`
	CanEnroll bool   `json:"canEnroll"`
}

type StudentRow struct {
	Id        string `json:"id"`
	BirthDate string `json:"birthDate,omitempty"`

	Position  uint16         `json:"position"`
	CanEnroll bool           `json:"canEnroll"`
	Courses   []CourseStatus `json:"courses"`

	Result          float32            `json:"result"`
	EnglishResult   uint8              `json:"englishResult,omitempty"`
	SectionsResults map[string]float32 `json:"sectionsResults"`
	Ofa             map[string]bool    `json:"ofa"`
}

type Ranking struct {
	Id     string `json:"id"`
	School string `json:"school"`
	Year   uint16 `json:"year"`

	// Stats    Stats
	Phase Phase        `json:"phase"`
	Rows  []StudentRow `json:"rows"`

	rowsById map[string]StudentRow
}

type RankingParser struct {
	rootDir string
	reader  writer.Writer[[]byte]
	Ranking Ranking

	mu sync.Mutex
}

func NewRankingParser(rootDir string) (*RankingParser, error) {
	ok, err := utils.DoFolderExists(rootDir)
	if !ok || err != nil {
		return nil, fmt.Errorf("Cannot create the RankingParser instance because the rootDir specified does not exist. rootDir: %s", rootDir)
	}

	reader, err := writer.NewWriter[[]byte](rootDir)
	if err != nil {
		return nil, err
	}

	parser := &RankingParser{rootDir: rootDir, reader: reader, Ranking: Ranking{}, mu: sync.Mutex{}}
	return parser, nil
}

func (p *RankingParser) Parse() *Ranking {
	splittedDir := strings.Split(p.rootDir, "/")
	p.Ranking.Id = splittedDir[len(splittedDir)-1]

	index, err := p.reader.Read(constants.OutputHtmlRanking_IndexFilename)
	if err != nil {
		slog.Error("Could not read Ranking index file", "filepath", path.Join(p.rootDir, constants.OutputHtmlRanking_IndexFilename), "error", err)
		return nil
	}

	err = p.parseIndex(index)
	if err != nil {
		slog.Error("Could not parse Ranking index file", "filepath", path.Join(p.rootDir, constants.OutputHtmlRanking_IndexFilename), "error", err)
		return nil
	}
	meritTablePages, err := utils.ReadAllFilesInFolder(path.Join(p.rootDir, constants.OutputHtmlRanking_ByMeritFolder))
	if err != nil {
		slog.Error("Could not read Ranking merit table file(s)", "folder-path", path.Join(p.rootDir, constants.OutputHtmlRanking_ByMeritFolder), "error", err)
		return nil
	}

	coursesTablePages, err := utils.ReadAllFilesInFolder(path.Join(p.rootDir, constants.OutputHtmlRanking_ByCourseFolder))
	if err != nil {
		slog.Error("Could not read Ranking course table file(s)", "folder-path", path.Join(p.rootDir, constants.OutputHtmlRanking_ByCourseFolder), "error", err)
		return nil
	}

	// IMPORTANT
	// run MERIT parser BEFORE COURSE parser
	//
	// MERIT
	err = p.parseMeritTable(meritTablePages)
	if err != nil {
		slog.Error("Could not parse Ranking merit table pages", "folder-path", path.Join(p.rootDir, constants.OutputHtmlRanking_ByMeritFolder), "error", err)
		return nil
	}
	//
	// COURSE
	err = p.parseAllCourseTables(coursesTablePages)
	if err != nil {
		slog.Error("Could not parse Ranking course table pages", "folder-path", path.Join(p.rootDir, constants.OutputHtmlRanking_ByCourseFolder), "error", err)
		return nil
	}

	return &p.Ranking
}

func (p *RankingParser) parseIndex(html []byte) error {
	doc, err := utils.LoadLocalHtml(html)
	if err != nil {
		return err
	}

	for i, s := range doc.Find(".CenterBar .intestazione").EachIter() {
		text, err := utils.GetFirstTextFragment(s)
		if err != nil {
			return err
		}
		lowerText := strings.ToLower(text)

		switch i {
		case 0:
			continue

		case 1:
			// year
			splitted := strings.Split(text, " ")
			yearSplitted := strings.Split(splitted[len(splitted)-1], "/")
			year, err := strconv.ParseUint(yearSplitted[0], 10, 16)
			if err != nil {
				return fmt.Errorf("Could not parse year. error: %w", err)
			}
			p.Ranking.Year = uint16(year)
			continue

		case 2:
			// language
			if strings.Contains(lowerText, "inglese") {
				p.Ranking.Phase.Language = constants.LangEn
			} else {
				p.Ranking.Phase.Language = constants.LangIt
			}

			// school
			if lowerText == "urbanistica: città ambiente paesaggio" {
				p.Ranking.School = constants.SchoolUrb
				continue
			}
			if strings.Contains(lowerText, "design") {
				p.Ranking.School = constants.SchoolDes
				continue
			}
			if strings.Contains(lowerText, "architettura") {
				p.Ranking.School = constants.SchoolArc
				continue
			}
			if strings.Contains(lowerText, "ingegneria") {
				p.Ranking.School = constants.SchoolIng
				continue
			}
			return fmt.Errorf("Could not parse school. school string: %s", text)

		case 3:
			p.Ranking.Phase.IsExtraEu = false
			err := p.Ranking.Phase.ParseText(text, &p.Ranking) // pass the non-lower version
			if err != nil {
				return fmt.Errorf("Could not parse phase. Phase raw string: '%s'. Error: %w", lowerText, err)
			}

		case 4:
			if strings.Contains(lowerText, "extra-ue") {
				p.Ranking.Phase.IsExtraEu = true
			}
			continue

		default:
			slog.Warn("Something is wrong with the index parsing, we got a 5-indexed element '.CenterBar .intestazione', maybe Polimi changed something. Please check.")
			continue
		}
	}

	return nil
}
