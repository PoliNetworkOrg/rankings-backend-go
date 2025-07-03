package parser

import (
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/scraper"
)

type (
	degreeMap   = map[string][]scraper.Manifesto
	locationMap = map[string]string
	courseMap   = map[string]locationMap
)

type ManifestiByDegreeType struct {
	DegreeType string    `json:"degreeType"`
	Data       courseMap `json:"data"`
}

type ManifestiByCourse struct {
	Data courseMap `json:"data"`
}

type RemoteManifesti struct {
	Data map[string](map[string](map[string]string)) // degree -> course name -> location -> url
}

func ParseManifestiByDegreeType(mans []scraper.Manifesto) []ManifestiByDegreeType {
	byDegType := groupByDegreeType(mans)
	out := make([]ManifestiByDegreeType, 0, len(byDegType))
	for dt, all := range groupByDegreeType(mans) {
		data := groupByCourse(all)
		m := ManifestiByDegreeType{DegreeType: dt, Data: data}
		out = append(out, m)
	}

	return out
}

func ParseManifestiByCourse(mans []scraper.Manifesto) ManifestiByCourse {
	byDegType := groupByCourse(mans)
	return ManifestiByCourse{
		Data: byDegType,
	}
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

func (m *ManifestiByDegreeType) GetAll() []scraper.Manifesto {
	out := make([]scraper.Manifesto, 0)
	for ck, m2 := range m.Data {
		for lk, url := range m2 {
			out = append(out, scraper.Manifesto{
				Name:       ck,
				Location:   lk,
				DegreeType: m.DegreeType,
				Url:        url,
			})
		}
	}

	return out
}

func (m *RemoteManifesti) ToList() []scraper.Manifesto {
	out := make([]scraper.Manifesto, 0)
	for degree, course_list := range m.Data {
		for course, location_list := range course_list {
			for location, url := range location_list {
				out = append(out, scraper.Manifesto{
					Name:       course,
					Location:   location,
					DegreeType: degree,
					Url:        url,
				})
			}
		}
	}

	return out
}
