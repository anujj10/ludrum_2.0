package parser

import "ludrum/internal/models"

func SanitizeOptions(options []models.OptionChain)([]models.OptionChain) {
	clean:= make([]models.OptionChain, 0, len(options))

	for _, opt:= range options {
		if opt.OptionType == "" {
			continue
		}

		if opt.StrikePrice <= 0 {
			continue
		}

		if opt.OI == 0 || opt.Volume == 0 {
			continue
		}

		clean= append(clean, opt)
	}
	return clean
}

func FilterNearATM(options []models.OptionChain, spot float64, strikerange float64) []models.OptionChain{
	atmOpt:= make([]models.OptionChain, 0)

	for _, opt:= range options {
		if opt.StrikePrice >= spot - strikerange && opt.StrikePrice <= spot + strikerange {
			atmOpt = append(atmOpt, opt)
		}
	}
	return atmOpt
}