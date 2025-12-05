package parser

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
	"github.com/PuerkitoBio/goquery"
)

func (p *RankingParser) parseAllCourseTables(pages [][]byte) error {
	// NOTE!!!
	// Run this function AFTER having parsed the merit table
	if len(p.Ranking.Rows) == 0 {
		return fmt.Errorf("This ranking does not have Merit table rows, so the course table is not parsed")
	}
	if p.Ranking.Rows[0].Id == "" {
		// considering this as expected, so no error returned
		slog.Warn("This ranking does not have Matricola IDs, so the course table is useless (we can't match data with merit table via the matricola id)", "id", p.Ranking.Id)
		return nil
	}

	wg := sync.WaitGroup{}
	errors := make([]string, 0)
	for _, page := range pages {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := p.parseCourseTable(page)
			if err != nil {
				errors = append(errors, err.Error())
			}
		}()
	}
	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("Error(s) during ranking table parsing:\n%s", strings.Join(errors, "\n"))
	}

	for id := range p.Ranking.rowsById {
		// sorting courses inside each student row
		slices.SortStableFunc(p.Ranking.rowsById[id].Courses, func(a, b CourseStatus) int {
			if a.Title < b.Title {
				return -1
			}
			if a.Title > b.Title {
				return 1
			}

			if a.Location < b.Location {
				return -1
			}
			if a.Location > b.Location {
				return 1
			}
			return 0
		})
	}

	newRows := slices.Collect(maps.Values(p.Ranking.rowsById))
	// sorting student rows by position in merit table
	slices.SortStableFunc(newRows, func(a, b StudentRow) int {
		if a.Position < b.Position {
			return -1
		}
		if a.Position > b.Position {
			return 1
		}
		return 0
	})

	p.Ranking.Rows = newRows
	return nil
}

func (p *RankingParser) parseCourseTable(html []byte) error {
	page, err := utils.LoadLocalHtml(html)
	if err != nil {
		return err
	}

	title, location := getCourseTitleLocation((page.Find(".CenterBar .titolo").First()).Text())
	slog := slog.With("ranking-id", p.Ranking.Id, "course-title", title, "course-location", location)
	c := CourseStatus{Title: title, Location: location}
	p.Ranking.addCourse(title, location)

	idIdx, birthIdx, posIdx, canEnrollIdx, engResultIdx, firstSectionIdx, ofaEngIdx, ofaTestIdx := -1, -1, -1, -1, -1, -1, -1, -1
	sections := make([]string, 0)

	for _, s := range page.Find(".TableDati tr:not(.elenco-campi) th").EachIter() {
		firstText, err := utils.GetFirstTextFragment(s)
		if err != nil {
			return err
		}

		sections = append(sections, firstText)
	}

	for i, s := range page.Find(".TableDati .elenco-campi th").EachIter() {
		firstText, err := utils.GetFirstTextFragment(s)
		if err != nil {
			return err
		}

		text := strings.ToLower(firstText)
		if strings.Contains(text, "sezioni") {
			firstSectionIdx = i
			continue
		}
		idx := i
		if firstSectionIdx != -1 && i > firstSectionIdx {
			idx += len(sections) - 1
		}
		if strings.Contains(text, "posizione") {
			posIdx = idx
			continue
		}
		if strings.Contains(text, "matricola") {
			idIdx = idx
			continue
		}
		if strings.Contains(text, "nascita") {
			birthIdx = idx
			continue
		}
		if strings.Contains(text, "consentita") {
			canEnrollIdx = idx
			continue
		}
		if strings.Contains(text, "risposte esatte inglese") {
			engResultIdx = idx
			continue
		}
		if strings.Contains(text, "ofa inglese") {
			ofaEngIdx = idx
			continue
		}
		if strings.Contains(text, "ofa test") {
			ofaTestIdx = idx
			continue
		}
	}

	for _, row := range page.Find(".TableDati-tbody tr").EachIter() {
		items := row.Find("td").Map(func(i int, s *goquery.Selection) string { return s.Text() })
		if len(items) == 1 && strings.Contains(items[0], "Nessun candidato") {
			slog.Debug("Course table is empty")
			continue
		}

		if len(items) == 0 {
			slog.Warn("Course table: <tr> contains 0 <td>, more in-depth investigation recommended")
			continue
		}

		if pos, err := strconv.ParseUint(p.getFieldByIndex(items, posIdx, "0"), 10, 16); err == nil {
			c.Position = uint16(pos)
		}

		rawId := p.getFieldByIndex(items, idIdx, "")
		id := strings.TrimSpace(strings.Replace(rawId, "(Contingente Marco Polo)", "", 1))
		if id == "" && p.Ranking.Year > 2020 {
			slog.Warn("Course table row without matricola ID", "position-in-table", c.Position)
		}
		if len(id) > 0 {
			id = utils.HashWithSalt(id)
		}

		p.mu.Lock()
		s := p.Ranking.rowsById[id] // student row parsed from merit table
		p.mu.Unlock()

		s.BirthDate = p.getFieldByIndex(items, birthIdx, "")

		if engResultIdx != -1 {
			engResultText := p.getFieldByIndex(items, engResultIdx, "-1")
			if engResult, err := strconv.ParseUint(engResultText, 10, 8); err == nil {
				s.EnglishResult = uint8(engResult)
			}
		}

		ofa := make(map[string]bool, 0)
		if ofaEngIdx != -1 {
			ofa["ENG"] = p.getFieldByIndex(items, ofaEngIdx, "No") != "No"
		}
		if ofaTestIdx != -1 {
			ofa["TEST"] = p.getFieldByIndex(items, ofaTestIdx, "No") != "No"
		}
		s.Ofa = ofa

		if canEnrollIdx != -1 {
			c.CanEnroll = p.getFieldByIndex(items, canEnrollIdx, "No") != "No"
		}

		if firstSectionIdx != -1 {
			sectionsResults := map[string]float32{}
			for i, section := range sections {
				idx := i + firstSectionIdx
				sectionText := strings.Replace(p.getFieldByIndex(items, idx, "-1"), ",", ".", 1)
				if sectionResult, err := strconv.ParseFloat(sectionText, 32); err == nil {
					sectionsResults[section] = float32(sectionResult)
				}
			}

			s.SectionsResults = sectionsResults
		}

		if s.Courses == nil {
			s.Courses = make([]CourseStatus, 0)
		}
		s.Courses = append(s.Courses, c)

		// save to map the modified data
		p.mu.Lock()
		p.Ranking.rowsById[id] = s // student row parsed from merit table
		p.mu.Unlock()
	}

	return nil
}

func (p *RankingParser) getFieldByIndex(items []string, index int, defaultValue string) string {
	if index == -1 {
		return defaultValue
	}

	if index > len(items)-1 {
		slog.Error("Error while parsing table: tried to index outside of row length", "ranking-id", p.Ranking.Id, "index", index, "row-length", len(items))
		return defaultValue
	}

	return strings.TrimSpace(items[index])
}

func getCourseTitleLocation(raw string) (string, string) {
	if strings.Contains(raw, "(") && strings.Contains(raw, ")") {
		// also with location
		splitted := strings.Split(raw, " (")
		locationSplitted := strings.Split(splitted[1], ")")
		return splitted[0], locationSplitted[0]
	} else {
		// without location
		return raw, ""
	}
}
