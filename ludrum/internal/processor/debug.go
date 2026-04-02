package processor

import (
	"fmt"
	// "log"
	"ludrum/internal/models"
)

func PrintAnalytics(data []models.StrikeAnalytics) {

	if len(data) == 0 {
		fmt.Println("No analytics yet (waiting for more ticks...)")
		return
	}

	fmt.Println("--------------------------------------------------")
	fmt.Println("Strike | Type | VolΔ | OIΔ | LTPΔ | Signal")
	fmt.Println("--------------------------------------------------")

	for _, a := range data {
		fmt.Printf("%.0f | %s | %d | %d | %.2f | %s\n",
			a.Strike,
			a.Type,
			a.VolumeChange,
			a.OIChange,
			a.LTPChange,
			a.Signal,
		)
	}
}