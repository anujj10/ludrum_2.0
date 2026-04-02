package processor

func detectSignal(ltpCh float64, oiCh int64) string {
	switch {
	case ltpCh > 0 && oiCh > 0:
		return "LONG_BUILDUP"

	case ltpCh < 0 && oiCh < 0:
		return "SHORT_BUILDUP"

	case ltpCh < 0 && oiCh > 0:
		return "SHORT_COVERING"

	case ltpCh < 0 && oiCh < 0:
		return "LONG_UNWINDING"

	default:
		return "NUETRAL"
	}
}