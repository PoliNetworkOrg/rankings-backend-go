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
	"github.com/PuerkitoBio/goquery"
)

type CourseStatus struct {
	Position  int    `json:"position"`
	CanEnroll bool   `json:"canEnroll"`
	Status    string `json:"status"`
}

type StudentRow struct {
	Id        string `json:"id"`
	BirthDate string `json:"birthDate,omitempty"`

	Position  uint16                  `json:"position"`
	CanEnroll bool                    `json:"canEnroll"`
	Courses   map[string]CourseStatus `json:"courses"`

	Result          float32            `json:"result"`
	EnglishResult   uint8              `json:"englishResult,omitempty"`
	SectionsResults map[string]float32 `json:"sectionsResults"`
	Ofa             map[string]bool    `json:"ofa"`
}

type Table struct {
	Headers []string     `json:"headers"`
	Rows    []StudentRow `json:"rows"`
}

type Ranking struct {
	Id     string `json:"id"`
	School string `json:"school"`
	Year   uint16 `json:"year"`

	// Stats    Stats
	Phase Phase `json:"phase"`
	Table Table `json:"table"`
}

type RankingParser struct {
	rootDir string
	reader  writer.Writer[[]byte]
	Ranking Ranking
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

	parser := &RankingParser{rootDir, reader, Ranking{}}
	return parser, nil
}

func (p *RankingParser) Parse() *Ranking {
	index, err := p.reader.Read(constants.OutputHtmlRanking_IndexFilename)
	if err != nil {
		slog.Error("Could not read Ranking index file", "filepath", path.Join(p.rootDir, constants.OutputHtmlRanking_IndexFilename), "error", err)
		return nil
	}

	splittedDir := strings.Split(p.rootDir, "/")
	p.Ranking.Id = splittedDir[len(splittedDir)-1]

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

	err = p.parseMeritTable(meritTablePages)
	if err != nil {
		slog.Error("Could not parse Ranking merit table pages", "folder-path", path.Join(p.rootDir, constants.OutputHtmlRanking_ByMeritFolder), "error", err)
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

func (p *RankingParser) parseMeritTable(pages [][]byte) error {
	wg := sync.WaitGroup{}
	table := &p.Ranking.Table
	errors := make([]string, 0)
	for i, page := range pages {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := p.parseMeritTablePage(page)
			if err != nil {
				errors = append(errors, err.Error())
			}

			if i == 0 {
				table.Headers = res.Headers
			}

			table.Rows = slices.Concat(table.Rows, res.Rows)
		}()
	}
	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("Error(s) during ranking table parsing:\n%s", strings.Join(errors, "\n"))
	}

	slices.SortStableFunc(table.Rows, func(a, b StudentRow) int {
		if a.Position < b.Position {
			return -1
		}
		if a.Position > b.Position {
			return 1
		}
		return 0
	})

	return nil
}

func (p *RankingParser) parseMeritTablePage(html []byte) (*Table, error) {
	page, err := utils.LoadLocalHtml(html)
	if err != nil {
		return nil, err
	}

	idIdx, resultIdx, posIdx, statusIdx, engResultIdx, ofaEngIdx, ofaTestIdx := -1, -1, -1, -1, -1, -1, -1

	table := &Table{Headers: make([]string, 0), Rows: make([]StudentRow, 0)}
	for i, s := range page.Find(".TableDati .elenco-campi th").EachIter() {
		firstText, err := utils.GetFirstTextFragment(s)
		if err != nil {
			return nil, err
		}

		text := strings.ToLower(firstText)

		table.Headers = append(table.Headers, firstText)
		if text == "matricola" {
			idIdx = i
			continue
		}
		if strings.Contains(text, "voto") {
			resultIdx = i
			continue
		}
		if strings.Contains(text, "posizione") {
			posIdx = i
			continue
		}
		if strings.Contains(text, "immatricolazione") || strings.Contains(text, "stato") {
			// immatricolazione --> ing, urb, des
			// stato --> arch
			statusIdx = i
			continue
		}
		if strings.Contains(text, "ofa inglese") {
			ofaEngIdx = i
			continue
		}
		if strings.Contains(text, "ofa test") {
			ofaTestIdx = i
			continue
		}
	}

	for _, row := range page.Find(".TableDati-tbody tr").EachIter() {
		s := StudentRow{}
		items := row.Find("td").Map(func(i int, s *goquery.Selection) string { return s.Text() })
		if len(items) == 0 {
			slog.Error("Error while parsing merit table, empty table row", "ranking-id", p.Ranking.Id)
			continue
		}

		if position, err := strconv.ParseUint(p.getFieldByIndex(items, posIdx, "0"), 10, 8); err == nil {
			s.Position = uint16(position)
		}

		s.Id = p.getFieldByIndex(items, idIdx, "")
		if s.Id == "" && p.Ranking.Year > 2020 {
			slog.Warn("Merit row without matricola ID", "ranking-id", p.Ranking.Id, "position", s.Position)
		}

		resultStr := p.getFieldByIndex(items, resultIdx, "0")
		if result, err := strconv.ParseFloat(strings.Replace(resultStr, ",", ".", 1), 32); err == nil {
			s.Result = float32(result)
		}

		if engResultIdx != -1 {
			engResultText := p.getFieldByIndex(items, engResultIdx, "-1")
			if engResult, err := strconv.ParseUint(engResultText, 10, 8); err == nil {
				s.EnglishResult = uint8(engResult)
			}
		}

		if position, err := strconv.ParseUint(p.getFieldByIndex(items, posIdx, "0"), 10, 16); err == nil {
			s.Position = uint16(position)
		}

		s.Ofa = make(map[string]bool, 0)
		if ofaEngIdx != -1 {
			s.Ofa["ENG"] = p.getFieldByIndex(items, ofaEngIdx, "No") != "No"
		}

		if ofaTestIdx != -1 {
			s.Ofa["TEST"] = p.getFieldByIndex(items, ofaTestIdx, "No") != "No"
		}

		// for the status we should make a little node
		// in some rankings the options are
		// - immatricolazione non consentita ...
		// - course name all uppercase
		// in other rankings the options are
		// - Attesa - immatricolazione non consentita ...
		// - Assegnato - course name all uppercase
		// - Prenotato - course name all uppercase
		// we should parse CanEnroll correctly
		//
		// another note:
		// - if we HAVE the Id, we fill the s.Courses field by parsing the course table
		// - if we DON'T HAVE the Id, we fill the s.Courses with the only course available from this table (obv only if the student can enroll)

		statusText := p.getFieldByIndex(items, statusIdx, "")
		if statusText == "" {
			slog.Warn("Merit row without status", "ranking-id", p.Ranking.Id, "position", s.Position)
		} else {
			lower := strings.ToLower(statusText)
			s.CanEnroll = !strings.Contains(lower, "immatricolazione non consentita / enrolment is not possible")

			if s.Id == "" && s.CanEnroll {
				// we DON'T HAVE the Id, we fill the s.Courses with the only course available from this table (obv only if the student can enroll)
				splitted := strings.Split(statusText, " - ")
				if len(splitted) == 2 {
					status := splitted[0]
					course := splitted[1]
					s.Courses[course] = CourseStatus{Status: status, CanEnroll: strings.ToLower(status) == "attesa"}
				} else {
					s.Courses[statusText] = CourseStatus{CanEnroll: true} // here statusText is the course name all uppercase
				}
			}
		}

		table.Rows = append(table.Rows, s)
	}

	return table, nil
}

func (p *RankingParser) getFieldByIndex(items []string, index int, defaultValue string) string {
	if index == -1 {
		return defaultValue
	}

	if index > len(items)-1 {
		slog.Error("Error while parsing table: tried to index outside of row length", "ranking-id", p.Ranking.Id, "index", index, "row-length", len(items))
		return defaultValue
	}

	return items[index]
}
