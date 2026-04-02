package postgres

import (
	"log"
	"time"

	"ludrum/internal/models"
)

// DBWorker now ONLY logs / processes analytics (no DB insert)
type DBWorker struct {
	PairChan chan []models.PairSignal
}

func NewDBWorker() *DBWorker {
	return &DBWorker{
		PairChan: make(chan []models.PairSignal, 100),
	}
}

func (w *DBWorker) Start() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)

		for {
			select {
			case pairs := <-w.PairChan:
				log.Println("Received pairs (analytics):", len(pairs))

			case <-ticker.C:
				// future: analytics persistence / feature store
			}
		}
	}()
}