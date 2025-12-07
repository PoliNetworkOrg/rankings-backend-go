package parser

import (
	"fmt"
	"slices"
	"sync"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/writer"
)

type indexEntry struct {
	ID     string `json:"id"`
	School string `json:"school"`
	Year   uint   `json:"year"`
	Phase  Phase  `json:"phase"`
}

type (
	bySchoolYear = map[string]map[uint][]indexEntry
	byYearSchool = map[uint]map[string][]indexEntry
	// bySchoolCourse = map[uint]map[string][]indexEntry // TODO
)

type IndexGenerator struct {
	outDir       string
	entries      []indexEntry
	byYearSchool byYearSchool
	bySchoolYear bySchoolYear
}

func NewIndexGenerator(absOutDir string) *IndexGenerator {
	return &IndexGenerator{
		outDir:       absOutDir,
		bySchoolYear: make(bySchoolYear),
		byYearSchool: make(byYearSchool),
	}
}

func (gen *IndexGenerator) Add(ranking *Ranking) {
	gen.entries = append(gen.entries, indexEntry{ID: ranking.Id, School: ranking.School, Year: uint(ranking.Year), Phase: ranking.Phase})
}

func (gen *IndexGenerator) makeSchoolYear() {
	for _, el := range gen.entries {
		// Ensure the inner map for the school exists
		if _, ok := gen.bySchoolYear[el.School]; !ok {
			gen.bySchoolYear[el.School] = make(map[uint][]indexEntry)
		}

		// Append the current indexEntry to the slice for the specific year
		gen.bySchoolYear[el.School][el.Year] = append(gen.bySchoolYear[el.School][el.Year], el)
	}

	for _, schoolMap := range gen.bySchoolYear {
		for year := range schoolMap {
			sortByPhases(schoolMap[year])
		}
	}
}

func (gen *IndexGenerator) makeYearSchool() {
	for _, el := range gen.entries {
		// Ensure the inner map for the school exists
		if _, ok := gen.byYearSchool[el.Year]; !ok {
			gen.byYearSchool[el.Year] = make(map[string][]indexEntry)
		}

		// Append the current indexEntry to the slice for the specific year
		gen.byYearSchool[el.Year][el.School] = append(gen.byYearSchool[el.Year][el.School], el)
	}

	for _, yearMap := range gen.byYearSchool {
		for school := range yearMap {
			sortByPhases(yearMap[school])
		}
	}
}

func (gen *IndexGenerator) Generate() error {
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		gen.makeSchoolYear()
	}()

	go func() {
		defer wg.Done()
		gen.makeYearSchool()
	}()

	wg.Wait()
	return gen.write()
}

func (gen *IndexGenerator) write() error {
	w1 := writer.NewWriter[bySchoolYear](gen.outDir)
	if err := w1.JsonWrite(constants.OutputIndexBySchoolYearFilename, gen.bySchoolYear, true); err != nil {
		return fmt.Errorf("error while performing write (1) in IndexGenerator, error: %w", err)
	}

	w2 := writer.NewWriter[byYearSchool](gen.outDir)
	if err := w2.JsonWrite(constants.OutputIndexByYearSchoolFilename, gen.byYearSchool, true); err != nil {
		return fmt.Errorf("error while performing write (1) in IndexGenerator, error: %w", err)
	}

	return nil
}

func sortByPhases(input []indexEntry) {
	slices.SortStableFunc(input, func(a, b indexEntry) int {
		return CmpPhases(a.Phase, b.Phase)
	})
}
