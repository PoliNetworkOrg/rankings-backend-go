package parser

import (
	"fmt"
	"strings"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
)

type Phase struct {
	Raw       string `json:"raw"`
	Primary   uint8  `json:"primary"`
	Secondary uint8  `json:"secondary"`
	Language  string `json:"language"`
	IsExtraEu bool   `json:"isExtraEu"`
}

func (p *Phase) ParseText(raw string, ranking *Ranking) error {
	// until 2023
	// arch --> 1 phase, multiple rankings (method 1)
	// ing --> 2-3 phases, multiple rankings per phase (method 2)
	// des, urb --> 1 phase, 3-4 rankings: Anticipato, Standard, Ripescaggio, Extra-ue (method 3)

	// since 2024 (until the last time this code was updated)
	// arch, des --> 1 phase, multiple rankings (method 1)
	// ing, urb --> 2-3 phases, multiple rankings per phase (method 2)

	//
	// so grouping by method
	// method 1 --> arch, des(2024)
	// method 2 --> ing, urb(2024)
	// method 3 --> des(2023), urb(2023)

	p.Raw = raw
	lower := strings.ToLower(raw)
	if ranking.School == constants.SchoolArc {
		return p.parseMethod1(lower) // ing
	}

	if ranking.School == constants.SchoolIng {
		return p.parseMethod2(lower) // arch
	}

	if ranking.School == constants.SchoolDes {
		if ranking.Year <= 2023 {
			return p.parseMethod3(lower) // des 2023
		}

		return p.parseMethod1(lower) // des 2024
	}

	if ranking.School == constants.SchoolUrb {
		if ranking.Year <= 2023 {
			return p.parseMethod3(lower) // urb 2023
		}

		return p.parseMethod2(lower) // urb 2024
	}

	return fmt.Errorf("Could not parse rankings phase, because there is no School")
}

func (p *Phase) parseMethod1(raw string) error {
	// one phase, multiple rankings per phase
	// extra eu information is not in this raw string (skipped)

	// seconda graduatoria 

	splitted := strings.Split(raw, " ")
	if len(splitted) != 2 {
		return fmt.Errorf("Invalid raw phase string, splitted by spaces array length is not 2. Raw: '%s' (method 1)", raw)
	}

	p.Primary = 0
	secondary := utils.GetOrdinalNumberInt(splitted[0])
	if secondary == 0 {
		return fmt.Errorf("Could not parse phase (secondary) number. Raw: '%s' (method 1)", raw)
	}

	p.Secondary = secondary
	return nil
}

func (p *Phase) parseMethod2(raw string) error {
	// multiple phases, multiple rankings per phase
	// extra eu information is not in this raw string (skipped)

	// seconda graduatoria di seconda fase

	splitted := strings.Split(raw, " ")
	if len(splitted) != 5 {
		return fmt.Errorf("Invalid raw phase string, splitted by spaces array length is not 5. Raw: '%s' (method 2)", raw)
	}

	secondary := utils.GetOrdinalNumberInt(splitted[0])
	primary := utils.GetOrdinalNumberInt(splitted[3])

	if primary == 0 || secondary == 0 {
		return fmt.Errorf("Could not parse phase number. Raw: '%s' (method 2)", raw)
	}

	p.Primary = primary
	p.Secondary = secondary

	return nil
}

func (p *Phase) parseMethod3(raw string) error {
	p.Primary = 0
	p.IsExtraEu = false
	if strings.Contains(raw, "anticipat") {
		p.Secondary = 1
	} else if strings.Contains(raw, "standard") {
		p.Secondary = 2
	} else if strings.Contains(raw, "extra-ue") {
		p.Secondary = 3
		p.IsExtraEu = true
	} else if strings.Contains(raw, "ripescaggio") {
		p.Secondary = 4
	} else {
		return fmt.Errorf("Could not parse before-2023 rankings phase. Raw: '%s' (method 3)", raw)
	}

	return nil
}
