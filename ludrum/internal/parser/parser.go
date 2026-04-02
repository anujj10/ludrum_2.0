package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"ludrum/internal/models"
)

func ParseOptionChain(data []byte) (*models.OptionChainResponse, error) {
	if len(data) == 0 {
		return nil, errors.New("empty response body")
	}

	var resp models.OptionChainResponse

	decoder:= json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	if err := decoder.Decode(&resp); err != nil{
		return nil, fmt.Errorf("json decode failed: %w", err)
	}

	if resp.Code  != 200 {
		return nil, fmt.Errorf("api error: code=%d message=%s", resp.Code, resp.Message)
	}

	if resp.Status != "ok" {
		return nil, fmt.Errorf("Options chain is empty")
	}

	for i, opt:= range resp.Data.OptionsChain{
		if opt.OptionType == "" {
			continue
		}

		if opt.StrikePrice < 0 {
			return nil, fmt.Errorf("invalid strike at index %d", i)
		}

		if opt.OptionType != "CE" && opt.OptionType != "PE" {
			return nil, fmt.Errorf("Invalid option type at index %d: %s", i, opt.OptionType)
		}
	} 
	

	return &resp, nil
}

