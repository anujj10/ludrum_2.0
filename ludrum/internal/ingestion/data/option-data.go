package optionData

import (
	"fmt"

	fyersgosdk "github.com/FyersDev/fyers-go-sdk"
)

func CreateFyersModel(appID, accessToken string) *fyersgosdk.FyersModel {
	fymodel := fyersgosdk.NewFyersModel(appID, accessToken)

	return fymodel
}

func OptionsData(symbol, timestamp string, strikeCount int, fymodel *fyersgosdk.FyersModel) (optionResp string, err error) {
	if fymodel == nil {
		return "", fmt.Errorf("fyers model is nil")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while fetching option chain: %v", r)
			optionResp = ""
		}
	}()

	optionResp, err = fymodel.GetOptionChain(fyersgosdk.OptionChainRequest{
		Symbol:      symbol,
		StrikeCount: strikeCount,
		Timestamp:   timestamp,
	})

	if err != nil {
		return "", fmt.Errorf("error getting option chain: %w", err)
	}

	if optionResp == "" {
		return "", fmt.Errorf("empty option chain response")
	}

	return optionResp, nil

}
