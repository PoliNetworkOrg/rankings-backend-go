package parser

import (
	"fmt"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/writer"
)

type checkPhase struct {
	ID     string `json:"id"`
	School string `json:"school"`
	Year   uint   `json:"year"`
	Phase  Phase  `json:"phase"`
}

type cpMap = map[string]map[uint][]checkPhase

type CheckPhases struct {
	outDir        string
	phases        []checkPhase
	groupedPhases cpMap
}

func NewCheckPhases(absOutDir string) CheckPhases {
	return CheckPhases{
		outDir:        absOutDir,
		phases:        make([]checkPhase, 0),
		groupedPhases: make(cpMap),
	}
}

func (cp *CheckPhases) Add(ranking *Ranking) {
	cp.phases = append(cp.phases, checkPhase{ID: ranking.Id, School: ranking.School, Year: uint(ranking.Year), Phase: ranking.Phase})
}

func (cp *CheckPhases) CreateGrouped() {
	for _, el := range cp.phases {
		// Ensure the inner map for the school exists
		if _, ok := cp.groupedPhases[el.School]; !ok {
			cp.groupedPhases[el.School] = make(map[uint][]checkPhase)
		}
		// Append the current CheckPhase to the slice for the specific year
		cp.groupedPhases[el.School][el.Year] = append(cp.groupedPhases[el.School][el.Year], el)
	}
}

func (cp *CheckPhases) Write() error {
	cp.CreateGrouped()

	w, err := writer.NewWriter[[]checkPhase](cp.outDir)
	if err != nil {
		return fmt.Errorf("error while creating writer (1) in CheckPhases, error: %w", err)
	}
	err = w.JsonWrite("phases.json", cp.phases, true)
	if err != nil {
		return fmt.Errorf("error while performing write (1) in CheckPhases, error: %w", err)
	}

	gw, err := writer.NewWriter[cpMap](cp.outDir)
	if err != nil {
		return fmt.Errorf("error while creating writer (2) in CheckPhases, error: %w", err)
	}
	err = gw.JsonWrite("phases_grouped.json", cp.groupedPhases, true)
	if err != nil {
		return fmt.Errorf("error while performing write (2) in CheckPhases, error: %w", err)
	}

	return nil
}
