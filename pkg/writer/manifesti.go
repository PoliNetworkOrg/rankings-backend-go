package writer

import (
	"encoding/json"
	"os"
	"path"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
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

func ManifestiFilePath(dataDir string) string {
	return path.Join(dataDir, constants.OutputBaseFolder, constants.OutputManifestiFilename)
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

func ReadManifestiJsonFile(dataDir string) ([]byte, error) {
	return os.ReadFile(ManifestiFilePath(dataDir))
}

func ParseManifestiJson(data []byte) (ManifestiJson, error) {
	var out ManifestiJson

	err := json.Unmarshal(data, &out)
	if err != nil {
		return NewManifestiJson([]scraper.Manifesto{}), err
	}

	return out, nil
}

func NewManifestiJson(mans []scraper.Manifesto) ManifestiJson {
	byDegType := groupByDegreeType(mans)
	data := make(map[string]courseMap, len(byDegType))
	for k, v := range byDegType {
		mapped := groupByCourse(v)
		data[k] = mapped
	}

	return ManifestiJson{Data: data}
}

func (m *ManifestiJson) Write(dataDir string) error {
	j, err := json.MarshalIndent(m.Data, "", "	")
	if err != nil {
		return err
	}

	err = os.WriteFile(ManifestiFilePath(dataDir), j, 0644)
	return err
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
