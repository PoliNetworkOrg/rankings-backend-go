package writer

import (
	"encoding/json"
	"os"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/scraper"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
)

type degreeMap = map[string][]scraper.Manifesto
type locationMap = map[string]string
type courseMap = map[string]locationMap

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

func WriteManifesti(mans []scraper.Manifesto) error {
	byDegType := groupByDegreeType(mans)
	out := make(map[string]courseMap, len(byDegType))
	for k,v := range byDegType {
		mapped := groupByCourse(v)
		out[k] = mapped
	}

	tmpExists, err := utils.DoFolderExists("tmp")
	if !tmpExists || err != nil {
		os.Mkdir("tmp", os.ModePerm)
	}

	jsonSlc, err := json.MarshalIndent(mans, "", "	")
	if err != nil {
		panic(err)
	}

	jsonMap, err := json.MarshalIndent(out, "", "	")
	if err != nil {
		panic(err)
	}

	os.WriteFile("tmp/test.json", jsonSlc, 0644)
	os.WriteFile("tmp/test_map.json", jsonMap, 0644)
	return nil
}
