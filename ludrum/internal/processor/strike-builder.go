package processor

import (
	"ludrum/internal/models"
)

func BuilderStrikeMap(options []models.OptionChain) map[float64]*models.StrikeData {
	strikemap:= make(map[float64]*models.StrikeData)


	for _, opt := range options{
		if opt.OptionType == "" && opt.StrikePrice <= 0 {
			continue
		}

		strike:= opt.StrikePrice

		//initialize if not exists
		if _, exists := strikemap[strike]; !exists {
			strikemap[strike]= &models.StrikeData{
				Strike: strike,
			}
		}
		
	//assign PE or CE
	optCopy := opt

	if opt.OptionType == "CE" {
		strikemap[strike].CE = &optCopy
	} else if opt.OptionType == "PE" {
		strikemap[strike].PE = &optCopy
	}
	}

	return strikemap
}
