package simulator

func (s *Simulator) checkExits(pos *Position, price float64) bool {

	if pos.SL != nil {
		if pos.Side == LONG && price <= *pos.SL {
			return true
		}
		if pos.Side == SHORT && price >= *pos.SL {
			return true
		}
	}

	if pos.Target != nil {
		if pos.Side == LONG && price >= *pos.Target {
			return true
		}
		if pos.Side == SHORT && price <= *pos.Target {
			return true
		}
	}

	return false
}