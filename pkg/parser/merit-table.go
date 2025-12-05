package parser

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
	"github.com/PuerkitoBio/goquery"
)

func (p *RankingParser) parseMeritTable(pages [][]byte) error {
	wg := sync.WaitGroup{}
	rows := make([]StudentRow, 0)
	errors := make([]string, 0)

	for _, page := range pages {
		wg.Add(1)
		go func() {
			defer wg.Done()
			newRows, err := p.parseMeritTablePage(page)
			if err != nil {
				errors = append(errors, err.Error())
			}

			rows = slices.Concat(rows, newRows)
		}()
	}
	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("Error(s) during ranking table parsing:\n%s", strings.Join(errors, "\n"))
	}

	slices.SortStableFunc(rows, func(a, b StudentRow) int {
		if a.Position < b.Position {
			return -1
		}
		if a.Position > b.Position {
			return 1
		}
		return 0
	})

	p.Ranking.Rows = rows
	return nil
}

func (p *RankingParser) parseMeritTablePage(html []byte) ([]StudentRow, error) {
	page, err := utils.LoadLocalHtml(html)
	if err != nil {
		return nil, err
	}

	idIdx, resultIdx, posIdx, statusIdx, ofaEngIdx, ofaTestIdx := -1, -1, -1, -1, -1, -1

	if p.Ranking.rowsById == nil {
		p.Ranking.rowsById = make(map[string]StudentRow, 0)
	}

	rows := make([]StudentRow, 0)
	for i, s := range page.Find(".TableDati .elenco-campi th").EachIter() {
		firstText, err := utils.GetFirstTextFragment(s)
		if err != nil {
			return nil, err
		}

		text := strings.ToLower(firstText)
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
		if s.Id == "" && p.Ranking.Year > 2021 {
			slog.Warn("Merit row without matricola ID", "ranking-id", p.Ranking.Id, "position", s.Position)
		}
		if len(s.Id) > 0 {
			s.Id = utils.HashWithSalt(s.Id)
		}

		resultStr := p.getFieldByIndex(items, resultIdx, "0")
		if result, err := strconv.ParseFloat(strings.Replace(resultStr, ",", ".", 1), 32); err == nil {
			s.Result = float32(result)
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

			if s.Courses == nil {
				s.Courses = make([]CourseStatus, 0)
			}

			if s.Id == "" && s.CanEnroll {
				// we DON'T HAVE the Id, we fill the s.Courses with the only course available from this table (obv only if the student can enroll)
				splitted := strings.Split(statusText, " - ")
				if len(splitted) == 2 {
					// "Assegnato - <course name>"
					course := splitted[1]
					title, location := getCourseTitleLocation(course)

					s.Courses = append(s.Courses, CourseStatus{Title: title, Location: location, CanEnroll: true})
				} else {
					// "<course name>"
					title, location := getCourseTitleLocation(statusText)
					s.Courses = append(s.Courses, CourseStatus{Title: title, Location: location, CanEnroll: true})
				}
			}
		}

		// save to Rows slice
		rows = append(rows, s)

		// save to rowsById map
		if s.Id != "" {
			p.mu.Lock()
			p.Ranking.rowsById[s.Id] = s
			p.mu.Unlock()
		}
	}

	return rows, nil
}
