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
		return &p.Ranking
	}

	return &p.Ranking
}

func (p *RankingParser) parseIndex(html []byte) error {
	doc, err := utils.LoadLocalHtml(html)
	if err != nil {
		return err
	}

	headings := make([]string, 5)
	for i, s := range doc.Find(".CenterBar .intestazione").EachIter() {
		text, err := utils.GetFirstTextFragment(s)
		if err != nil {
			return err
		}

		if i >= 5 {
			slog.Warn("Something is wrong with the index parsing, we got a 5-indexed element '.CenterBar .intestazione', maybe Polimi changed something. Please check", "heading index", i, "text", text)
			break
		}

		headings[i] = text
	}

	if err = p.Ranking.parseYear(headings[1]); err != nil {
		return err
	}

	if err = p.Ranking.parseSchoolLang(headings[2]); err != nil {
		return err
	}

	p.Ranking.Phase.IsExtraEu = strings.Contains(strings.ToLower(headings[4]), "extra-ue")

	if err = p.Ranking.Phase.ParseText(headings[3], &p.Ranking); err != nil {
		return fmt.Errorf("Could not parse phase. Phase raw string: '%s'. Error: %w", strings.ToLower(headings[3]), err)
	}

	return nil
}

func (r *Ranking) parseYear(s string) error {
	// year
	splitted := strings.Split(s, " ")
	yearSplitted := strings.Split(splitted[len(splitted)-1], "/")
	year, err := strconv.ParseUint(yearSplitted[0], 10, 16)
	if err != nil {
		return fmt.Errorf("Could not parse year. error: %w", err)
	}

	r.Year = uint16(year)
	return nil
}

func (r *Ranking) parseSchoolLang(s string) error {
	lower := strings.ToLower(s)
	// language
	if strings.Contains(lower, "inglese") {
		r.Phase.Language = constants.LangEn
	} else {
		r.Phase.Language = constants.LangIt
	}

	// school
	if strings.Contains(lower, "urbanistica") {
		r.School = constants.SchoolUrb
	} else if strings.Contains(lower, "design") {
		r.School = constants.SchoolDes
	} else if strings.Contains(lower, "architettura") {
		r.School = constants.SchoolArc
	} else if strings.Contains(lower, "ingegneria") {
		r.School = constants.SchoolIng
	} else {
		return fmt.Errorf("Could not parse school. school string: %s", s)
	}

	return nil
}
