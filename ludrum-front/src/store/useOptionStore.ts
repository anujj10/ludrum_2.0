import { create } from "zustand"
import { persist } from "zustand/middleware"
import type { PairSignal, Portfolio, Position, StreamData } from "../types/option"

type State = {
  strikeMap: Record<number, PairSignal>
  spot: number
  openPositions: Position[]
  closedPositions: Position[]
  portfolio: Portfolio | null
  lastType: "snapshot" | "delta" | null
  hydrate: (payload: StreamData, mode: "snapshot" | "delta") => void
}

const EMPTY_PORTFOLIO: Portfolio = {
  InitialCapital: 0,
  AvailableCapital: 0,
  UsedMargin: 0,
  RealizedPnL: 0,
  UnrealizedPnL: 0,
}

function latestLtp(leg: PairSignal["CE"] | PairSignal["PE"] | undefined): number | undefined {
  const series = leg?.LTPSeries
  if (!series?.length) return undefined
  return series[series.length - 1]
}

function hasMeaningfulLegData(leg: PairSignal["CE"] | PairSignal["PE"] | undefined) {
  if (!leg) return false

  return Boolean(
    leg.Type ||
      leg.CurrentOI ||
      leg.OIChange ||
      leg.LTPChange ||
      leg.VolumeChange ||
      leg.LTPSeries?.length ||
      leg.LTPDeltas?.length ||
      leg.LTPPattern?.length,
  )
}

function mergeLeg(
  existing: PairSignal["CE"] | PairSignal["PE"] | undefined,
  incoming: PairSignal["CE"] | PairSignal["PE"] | undefined,
) {
  if (!hasMeaningfulLegData(incoming)) {
    if (existing) return existing
    return incoming as PairSignal["CE"] | PairSignal["PE"]
  }

  if (!existing) return incoming as PairSignal["CE"] | PairSignal["PE"]

  const next = incoming as PairSignal["CE"] | PairSignal["PE"]

  return {
    ...existing,
    ...next,
    LTPSeries: next.LTPSeries?.length ? next.LTPSeries : existing.LTPSeries,
    LTPDeltas: next.LTPDeltas?.length ? next.LTPDeltas : existing.LTPDeltas,
    LTPPattern: next.LTPPattern?.length ? next.LTPPattern : existing.LTPPattern,
  }
}

function mergePair(existing: PairSignal | undefined, incoming: PairSignal): PairSignal {
  if (!existing) return incoming

  return {
    ...existing,
    ...incoming,
    CE: mergeLeg(existing.CE, incoming.CE),
    PE: mergeLeg(existing.PE, incoming.PE),
  }
}

export const useOptionStore = create<State>()(
  persist(
    (set) => ({
      strikeMap: {},
      spot: 0,
      openPositions: [],
      closedPositions: [],
      portfolio: EMPTY_PORTFOLIO,
      lastType: null,

      hydrate: (payload, mode) => {
        set((state) => {
          const nextMap = { ...state.strikeMap }

          for (const pair of payload.pairs ?? []) {
            const strike = Number(pair.Strike)
            if (!strike) continue

            const previousPair = state.strikeMap[strike]
            nextMap[strike] = mergePair(nextMap[strike], pair)
            const mergedPair = nextMap[strike]
            const ceOiDelta =
              previousPair?.CE?.CurrentOI !== undefined
                ? mergedPair.CE.CurrentOI - previousPair.CE.CurrentOI
                : undefined
            const peOiDelta =
              previousPair?.PE?.CurrentOI !== undefined
                ? mergedPair.PE.CurrentOI - previousPair.PE.CurrentOI
                : undefined

            nextMap[strike] = {
              ...mergedPair,
              CE: {
                ...mergedPair.CE,
                OIChange: ceOiDelta ?? mergedPair.CE.OIChange,
                LTPChange:
                  previousPair?.CE
                    ? (latestLtp(mergedPair.CE) ?? 0) - (latestLtp(previousPair.CE) ?? 0)
                    : mergedPair.CE.LTPChange,
              },
              PE: {
                ...mergedPair.PE,
                OIChange: peOiDelta ?? mergedPair.PE.OIChange,
                LTPChange:
                  previousPair?.PE
                    ? (latestLtp(mergedPair.PE) ?? 0) - (latestLtp(previousPair.PE) ?? 0)
                    : mergedPair.PE.LTPChange,
              },
            }
          }

          return {
            strikeMap: nextMap,
            spot: payload.spot ?? state.spot,
            openPositions: payload.open_positions ?? state.openPositions,
            closedPositions: payload.closed_positions ?? state.closedPositions,
            portfolio: payload.portfolio ?? state.portfolio,
            lastType: mode,
          }
        })
      },
    }),
    {
      name: "ludrum-terminal-store",
      partialize: (state) => ({
        strikeMap: state.strikeMap,
        spot: state.spot,
      }),
    },
  ),
)
