package parser

import (
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/scraper"
)

type (
	degreeMap   = map[string][]scraper.Manifesto
	locationMap = map[string]string
	courseMap   = map[string]locationMap
)

type ManifestiJson struct {
	Data map[string]courseMap // json output structure
}

func groupByDegreeType(mans []scraper.Manifesto) degreeMap {
	out := make(degreeMap, len(mans))

	for _, m := range mans {
		if out[m.DegreeType] == nil {
			out[m.DegreeType] = make([]scraper.Manifesto, 0)
		}
		out[m.DegreeType] = append(out[m.DegreeType], m)
	}

	return out
}

func groupByCourse(mans []scraper.Manifesto) courseMap {
	out := make(courseMap, len(mans))

	for _, m := range mans {
		if out[m.Name] == nil {
			out[m.Name] = make(locationMap)
		}
		out[m.Name][m.Location] = m.Url
	}

	return out
}

func (m *ManifestiJson) GetSlice() []scraper.Manifesto {
	out := make([]scraper.Manifesto, 0)
	for dtk, m1 := range m.Data {
		for ck, m2 := range m1 {
			for lk, url := range m2 {
				out = append(out, scraper.Manifesto {
					Name: ck,
					Location: lk,
					DegreeType: dtk,
					Url: url,
				})
			}
		}
	}

	return out
}
