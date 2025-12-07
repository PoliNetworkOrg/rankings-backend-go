package parser

import (
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/writer"
)

type IdHashIndexParser struct {
	outDir string
	index  map[string][]string
	mu     sync.Mutex
}

func NewIdHashIndexParser(absOutDir string) *IdHashIndexParser {
	return &IdHashIndexParser{
		outDir: absOutDir,
		index:  map[string][]string{},
		mu:     sync.Mutex{},
	}
}

func (p *IdHashIndexParser) Add(ranking *Ranking) {
	ids := maps.Keys(ranking.rowsById)
	for id := range ids {
		rankings := []string{ranking.Id}
		p.mu.Lock()

		if prev, found := p.index[id]; found {
			rankings = slices.Concat(rankings, prev)
		}

		p.index[id] = rankings
		p.mu.Unlock()
	}
}

func (p *IdHashIndexParser) Write() error {
	w := writer.NewWriter[map[string][]string](p.outDir)
	err := w.JsonWrite("studentIdHashIndex.json", p.index, true)
	if err != nil {
		return fmt.Errorf("error while performing write (1) in IdHashIndexParser, error: %w", err)
	}

	return nil
}
