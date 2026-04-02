package payloadbuilder

func (b *Builder) buildRisk(trade TradeDecision) RiskBlock {

	riskAmount := b.capital * b.riskPct

	perUnitRisk := trade.Entry - trade.SL
	if perUnitRisk <= 0 {
		return RiskBlock{}
	}

	qty := int(riskAmount / perUnitRisk)

	return RiskBlock{
		Capital:      b.capital,
		RiskPerTrade: b.riskPct,
		PositionSize: qty,
	}
}