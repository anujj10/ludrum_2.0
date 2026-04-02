package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type TradeRequest struct {
	Symbol     string  `json:"symbol"`
	Strike     float64 `json:"strike"`
	OptionType string  `json:"optionType"`
	Side       string  `json:"side"`
	Lots       int     `json:"lots"`
}

type ExitRequest struct {
	Strike     float64 `json:"strike"`
	OptionType string  `json:"optionType"`
}

type Position struct {
	Strike        float64 `json:"strike"`
	OptionType    string  `json:"optionType"`
	Qty           int     `json:"qty"`
	AvgPrice      float64 `json:"avg_price"`
	Side          string  `json:"side"`
	UnrealizedPnL float64 `json:"unrealizedPnL"`
}

type Portfolio struct {
	AvailableCapital float64 `json:"availableCapital"`
	UsedMargin       float64 `json:"usedMargin"`
	RealizedPnL      float64 `json:"realizedPnL"`
}

type PositionsResponse struct {
	Positions []Position `json:"positions"`
	Portfolio Portfolio  `json:"portfolio"`
}

func main() {
	for {
		var action string

		fmt.Println("\nEnter action: BUY / SELL / EXIT / POSITIONS")
		fmt.Scanln(&action)

		action = strings.ToUpper(action)

		switch action {

		// ================= BUY / SELL =================
		case "BUY", "SELL":

			var strike float64
			fmt.Println("Strike:")
			fmt.Scanln(&strike)

			var opt string
			fmt.Println("Option Type (CE/PE):")
			fmt.Scanln(&opt)
			opt = strings.ToUpper(opt)

			var lots int
			fmt.Println("Lots:")
			fmt.Scanln(&lots)

			req := TradeRequest{
				Symbol:     "NIFTY",
				Strike:     strike,
				OptionType: opt,
				Side:       action,
				Lots:       lots,
			}

			body, _ := json.Marshal(req)

			resp, err := http.Post(
				"http://localhost:8082/trade",
				"application/json",
				bytes.NewBuffer(body),
			)

			if err != nil {
				log.Println("❌ Error:", err)
				continue
			}

			fmt.Println("✅ Trade Status:", resp.Status)

		// ================= EXIT =================
		case "EXIT":

			var strike float64
			fmt.Println("Strike:")
			fmt.Scanln(&strike)

			var opt string
			fmt.Println("Option Type (CE/PE):")
			fmt.Scanln(&opt)
			opt = strings.ToUpper(opt)

			req := ExitRequest{
				Strike:     strike,
				OptionType: opt,
			}

			body, _ := json.Marshal(req)

			resp, err := http.Post(
				"http://localhost:8082/exit",
				"application/json",
				bytes.NewBuffer(body),
			)

			if err != nil {
				log.Println("❌ Error:", err)
				continue
			}

			fmt.Println("✅ Exit Status:", resp.Status)

		// ================= POSITIONS =================
		case "POSITIONS":

			resp, err := http.Get("http://localhost:8082/positions")
			if err != nil {
				log.Println("❌ Error:", err)
				continue
			}

			var data PositionsResponse
			err = json.NewDecoder(resp.Body).Decode(&data)
			if err != nil {
				log.Println("❌ Decode Error:", err)
				continue
			}

			fmt.Println("\n========== OPEN POSITIONS ==========")

			if len(data.Positions) == 0 {
				fmt.Println("No open positions")
			}

			for _, p := range data.Positions {
				fmt.Printf(
					"%s %s | Qty:%d | Avg:%.2f | PnL: %.2f\n",
					p.OptionType,
					sideLabel(p.Side),
					p.Qty,
					p.AvgPrice,
					p.UnrealizedPnL,
				)
			}

			fmt.Println("-----------------------------------")
			fmt.Printf("Capital: %.2f | Used: %.2f | Realized: %.2f\n",
				data.Portfolio.AvailableCapital,
				data.Portfolio.UsedMargin,
				data.Portfolio.RealizedPnL,
			)

		default:
			fmt.Println("❌ Invalid action")
		}
	}
}

func sideLabel(s string) string {
	if s == "LONG" {
		return "BUY"
	}
	return "SELL"
}