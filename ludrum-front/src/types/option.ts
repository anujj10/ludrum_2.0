export type SignalSide = "LONG" | "SHORT" | "HOLD" | "EXIT" | "NEUTRAL" | string

export type StrikeAnalytics = {
  Strike: number
  Type: string
  VolumeChange: number
  OIChange: number
  LTPChange: number
  CurrentOI: number
  LTPSeries: number[]
  LTPDeltas: number[]
  LTPPattern: string[]
  Velocity: number
  Acceleration: number
  OIMomentum: number
  VolumeSpike: number
  Signal: SignalSide
  Highlight: boolean
}

export type PairSignal = {
  Strike: number
  CE: StrikeAnalytics
  PE: StrikeAnalytics
  Bias: string
  Score: number
  Strength: string
}

export type Portfolio = {
  InitialCapital: number
  AvailableCapital: number
  UsedMargin: number
  RealizedPnL: number
  UnrealizedPnL: number
}

export type Position = {
  Symbol: string
  Strike: number
  OptionType: string
  Qty: number
  AvgPrice: number
  Side: "LONG" | "SHORT"
  SL?: number | null
  Target?: number | null
  UnrealizedPnL: number
  RealizedPnL: number
  EntryTime: number
  LastUpdate: number
}

export type OIChangeEvent = {
  time: string
  symbol: string
  strike: number
  option_type: "CE" | "PE"
  oi_change: number
  ltp_change: number
}

export type StreamData = {
  spot?: number
  pairs?: PairSignal[]
  open_positions?: Position[]
  closed_positions?: Position[]
  portfolio?: Portfolio
}

export type StreamMessage = {
  type: "snapshot" | "delta"
  data: StreamData
}
