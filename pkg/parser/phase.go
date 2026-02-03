package parser

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/constants"
	"github.com/PoliNetworkOrg/rankings-backend-go/pkg/utils"
)

type Phase struct {
	Raw       string `json:"raw"`
	Stripped  string `json:"stripped"`
	Primary   uint8  `json:"primary"`
	Secondary uint8  `json:"secondary"`
	Language  string `json:"language"`
	IsExtraEu bool   `json:"isExtraEu"`
}

func stripSchoolPrefix(s string) string {
	s, _ = strings.CutPrefix(s, "Architettura - ")
	s, _ = strings.CutPrefix(s, "Urbanistica - ")
	s, _ = strings.CutPrefix(s, "Design - ")
	s, _ = strings.CutPrefix(s, "Ingegneria - ")
	return s
}

func (p *Phase) ParseText(raw string, ranking *Ranking) error {
	slog := slog.With("raw", raw, "extra-eu", p.IsExtraEu)
	slog.Debug("--- STARTING PHASE PARSING ---")
	p.Raw = raw
	// it was common to add a prefix to indicate the School
	// (I mean, wtf, u already said that in the above heading, but they don't care)
	// so we strip it
	schoolStrip := stripSchoolPrefix(raw)
	slog.Debug("raw phase, stripped by school prefix", "schoolStrip", schoolStrip)

	// sometimes they alos add a prefix to indicate extra-eu
	// (again, wtf, u already said that in the following heading, but they don't care)
	// so we strip it
	// WARNING: the prefix is "Extra-ue - [...other things...]", DO NOT confuse with "Extra-ue [Prima ...]"
	// the prefix INCLUDES " - "
	eeuStrip, ok := strings.CutPrefix(schoolStrip, "Extra-ue - ")
	if ok {
		slog.Debug("raw phase, stripped by extra-ue prefix", "raw", raw, "schoolStrip", schoolStrip, "extra-eu strip", eeuStrip)
	} else {
		slog.Debug("raw phase, NOT HAVE extra-ue prefix", "raw", raw, "schoolStrip", schoolStrip, "extra-eu strip", eeuStrip)
	}

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

	p.Stripped = eeuStrip
	lower := strings.ToLower(eeuStrip)
	if ranking.School == constants.SchoolArc {
		return p.parseMethod1(lower) // ing
	}

	if ranking.School == constants.SchoolIng {
		return p.parseMethod2(lower, ranking.School) // arch
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

		return p.parseMethod2(lower, ranking.School) // urb 2024
	}

	return fmt.Errorf("Could not parse rankings phase, because there is no School")
}

func (p *Phase) parseMethod1(lower string) error {
	switch lower {
	// some 2020 old arch stuff
	case "extra-ue":
		p.IsExtraEu = true
		p.Secondary = 1
		return nil
	case "extra-ue - seconda graduatoria":
		p.IsExtraEu = true
		p.Secondary = 2
		return nil
	}

	// one phase, multiple rankings per phase
	// extra eu information is not in this raw string (skipped)
	//
	// seconda graduatoria

	splitted := strings.Split(lower, " ")
	if len(splitted) != 2 {
		return fmt.Errorf("Invalid raw phase string, splitted by spaces array length is not 2. Raw: '%s' (method 1)", lower)
	}

	number := utils.GetOrdinalNumberInt(splitted[0])
	if number == 0 {
		return fmt.Errorf("Could not parse phase (secondary) number. Raw: '%s' (method 1)", lower)
	}

	if splitted[1] == "fase" {
		// es. terza fase (see ing 2020)
		p.Primary = number
		p.Secondary = 0
	} else {
		p.Primary = 0
		p.Secondary = number
	}

	return nil
}

func (p *Phase) parseMethod2(lower string, school string) error {
	// EXTRA-EU
	if p.IsExtraEu {
		if school == constants.SchoolUrb {
			// some urbanistica new stuff
			if strings.Contains(lower, "anticipat") {
				p.Secondary = 1
			} else if strings.Contains(lower, "standard") {
				p.Secondary = 2
			} else if strings.Contains(lower, "ripescaggio") {
				p.Secondary = 3
			} else {
				return fmt.Errorf("[URBANISTICA] Could not parse extra-eu post-2024 rankings phase. Raw: '%s' (method 2)", lower)
			}

			return nil
		}

		if school == constants.SchoolArc {
			slog.Debug("extra-eu arc not handled (method 2)", "lower", lower)
			panic("NOT HANDLED")
		}

		if school == constants.SchoolIng {
			switch lower {
			// some 2020 old ing stuff
			case "extra-ue":
				p.Primary = 0
				p.Secondary = 1
				return nil
			case "extra-ue - seconda graduatoria":
				p.Primary = 0
				p.Secondary = 2
				return nil
			case "extra-ue - terza graduatoria":
				p.Primary = 0
				p.Secondary = 3
				return nil
			}

			// here we can get some phases like
			// Prima Graduatoria Di Seconda Fase
			// even if this is extra eu
			// so we pass it to the following non extra-eu phase
			lower, _ = strings.CutPrefix(lower, "extra-ue")
		}
	}

	// NON EXTRA-EU or ING EXTRA-EU stripped
	splitted := strings.Split(lower, " ")
	if len(splitted) == 2 {
		// fallback to method 1
		// there are some 2020 rankings that are so inconsistent
		return p.parseMethod1(lower)
	}

	// multiple phases, multiple rankings per phase
	// extra eu information is not in this raw string (skipped)
	//
	// seconda graduatoria di seconda fase

	if len(splitted) != 5 {
		return fmt.Errorf("Invalid raw phase string, splitted by spaces array length is not 5. Raw: '%s' (method 2)", lower)
	}

	secondary := utils.GetOrdinalNumberInt(splitted[0])
	primary := utils.GetOrdinalNumberInt(splitted[3])

	if primary == 0 || secondary == 0 {
		return fmt.Errorf("Could not parse phase number. Raw: '%s' (method 2)", lower)
	}

	p.Primary = primary
	p.Secondary = secondary

	return nil
}

func (p *Phase) parseMethod3(lower string) error {
	p.Primary = 0
	p.IsExtraEu = false
	if strings.Contains(lower, "anticipat") {
		p.Secondary = 1
	} else if strings.Contains(lower, "standard") {
		p.Secondary = 2
	} else if strings.Contains(lower, "extra-ue") {
		p.Secondary = 1
		p.IsExtraEu = true
	} else if strings.Contains(lower, "ripescaggio") {
		p.Secondary = 3
	} else {
		return fmt.Errorf("Could not parse before-2023 rankings phase. Raw: '%s' (method 3)", lower)
	}

	return nil
}
