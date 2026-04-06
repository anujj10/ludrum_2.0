package postgres

import "time"

func CurrentMarketSessionStartUTC(now time.Time) time.Time {
	kolkata, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return now.UTC()
	}

	localNow := now.In(kolkata)
	sessionStartLocal := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 9, 15, 0, 0, kolkata)
	return sessionStartLocal.UTC()
}
