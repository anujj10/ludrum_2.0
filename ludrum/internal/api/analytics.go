package api

// import (
// 	"encoding/json"
// 	processor "ludrum/internal/processor/analytics-engine"
// 	"net/http"
// )

// func AnalyticsHandler(p *processor.Pipeline) http.HandlerFunc{
// 	return func(w http.ResponseWriter, r *http.Request){
// 		json.NewEncoder(w).Encode(p.LatestPairs)
// 	}
// }