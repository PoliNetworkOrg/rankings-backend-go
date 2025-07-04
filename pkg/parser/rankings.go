package parser

import (
	"fmt"
	"log/slog"
	"path"
	"strconv"
	"strings"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/writer"
)

type CourseStatus struct {
	Position  int
	CanEnroll bool
	Status    string
}

type StudentRow struct {
	Id        string
	BirthDate string

	Position  int
	CanEnroll bool
	Courses   map[string]CourseStatus

	Result          float32
	EnglishResult   int
	SectionsResults map[string]float32
	Ofa             map[string]bool
}

type Table struct {
	Headers []string
	Rows    []StudentRow
}

type Ranking struct {
	School string
	Year   uint16
	Course string

	// Stats    Stats
	Phase Phase
	Table Table
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

	err = p.parseIndex(index)
	if err != nil {
		slog.Error("Could not parse Ranking index file", "filepath", path.Join(p.rootDir, constants.OutputHtmlRanking_IndexFilename), "error", err)
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
		html, err := s.Html()
		if err != nil {
			panic(err)
		}
		splittedHtml := strings.Split(html, "<br/>") // they love <br/> to separate languages
		text := strings.ToLower(splittedHtml[0])

		switch i {
		case 0:
			continue

		case 1:
			// year
			splitted := strings.Split(text, " ")
			yearSplitted := strings.Split(splitted[len(splitted)-1], "/")
			year, err := strconv.ParseUint(yearSplitted[0], 10, 16)
			if err != nil {
				return fmt.Errorf("Could not parse year. error: %v", err)
			}
			p.Ranking.Year = uint16(year)
			continue

		case 2:
			// language
			if strings.Contains(text, "inglese") {
				p.Ranking.Phase.Language = constants.LangEn
			} else {
				p.Ranking.Phase.Language = constants.LangIt
			}

			// school
			if text == "urbanistica: città ambiente paesaggio" {
				p.Ranking.School = constants.SchoolUrb
				continue
			}
			if strings.Contains(text, "design") {
				p.Ranking.School = constants.SchoolDes
				continue
			}
			if strings.Contains(text, "architettura") {
				p.Ranking.School = constants.SchoolArc
				continue
			}
			if strings.Contains(text, "ingegneria") {
				p.Ranking.School = constants.SchoolIng
				continue
			}
			return fmt.Errorf("Could not parse school. school string: %s", text)

		case 3:
			p.Ranking.Phase.IsExtraEu = false
			err := p.Ranking.Phase.ParseText(splittedHtml[0], &p.Ranking) // pass the non-lower version
			if err != nil {
				return fmt.Errorf("Could not parse phase. Phase raw string: '%s'. Error: %v", text, err)
			}

		case 4:
			if strings.Contains(text, "extra-ue") {
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
