package ltpSeries

import (
	"fmt"
	"log"

	fyersgosdk "github.com/FyersDev/fyers-go-sdk"

	"ludrum/internal/models"
	"ludrum/internal/parser"
)

type FyersFetcher struct {
	fyModel *fyersgosdk.FyersModel
}

func NewFyersFetcher(fyModel *fyersgosdk.FyersModel) *FyersFetcher {
	return &FyersFetcher{
		fyModel: fyModel,
	}
}

func (f *FyersFetcher) FetchOptionChain() (options []models.OptionChain, err error) {
	if f == nil || f.fyModel == nil {
		return nil, fmt.Errorf("fyers fetcher is not initialized")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while fetching option chain: %v", r)
			options = nil
		}
	}()

	resp, err := f.fyModel.GetOptionChain(fyersgosdk.OptionChainRequest{
		Symbol:      "NSE:NIFTY50-INDEX",
		StrikeCount: 2,
		Timestamp:   "",
	})

	if err != nil {
		log.Println("❌ Fetch error:", err)
		return nil, err
	}

	// 🔥 CRITICAL FIX
	if resp == "" {
		return nil, fmt.Errorf("empty response from API")
	}

	parsed, err := parser.ParseOptionChain([]byte(resp))
	if err != nil {
		log.Println("❌ Parse error:", err)
		return nil, err
	}

	return parsed.Data.OptionsChain, nil
}
