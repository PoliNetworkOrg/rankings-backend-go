package parser

import (
	"fmt"
	"log/slog"
	"path"
	"slices"
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

	Position uint16 `json:"position"`

	// if CanEnroll is true, we have two options:
	// 1. we have Id
	//		in this case, we have all the student's subscribed courses linked
	//		but one and only one has CanEnroll field true, so we get it from there
	// 2. we don't have Id
	//		in this case, we only have the course in which the student is allowed to be enrolled
	//
	// if CanEnroll is false, we dont give a fuck
	CanEnroll bool `json:"canEnroll"`

	Courses []CourseStatus `json:"courses"`

	Result          float32            `json:"result"`
	EnglishResult   uint8              `json:"englishResult,omitempty"`
	SectionsResults map[string]float32 `json:"sectionsResults"`
	Ofa             map[string]bool    `json:"ofa"`
}

type Ranking struct {
	Id     string `json:"id"`
	School string `json:"school"`
	Year   uint16 `json:"year"`

	// Stats   Stats
	Phase   Phase               `json:"phase"`
	Courses map[string][]string `json:"courses"`
	Rows    []StudentRow        `json:"rows"`

	rowsById map[string]StudentRow
}

type RankingParser struct {
	rootDir string
	reader  writer.Writer[[]byte]
	Ranking Ranking

	mu sync.Mutex
}

func NewRanking() *Ranking {
	return &Ranking{
		rowsById: map[string]StudentRow{},
		Courses:  map[string][]string{},
	}
}

func NewRankingParser(rootDir string) *RankingParser {
	reader := writer.NewWriter[[]byte](rootDir)
	return &RankingParser{rootDir: rootDir, reader: reader, Ranking: *NewRanking(), mu: sync.Mutex{}}
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

var rankingCoursesMutex = sync.Mutex{}

func (r *Ranking) addCourse(title, location string) {
	locations := []string{}
	rankingCoursesMutex.Lock()
	if prev, exists := r.Courses[title]; exists {
		locations = slices.Concat(locations, prev)
	}

	if len(location) > 0 {
		locations = append(locations, location)
	}

	r.Courses[title] = locations
	rankingCoursesMutex.Unlock()
}
