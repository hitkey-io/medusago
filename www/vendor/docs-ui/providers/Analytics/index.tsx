"use client"

// MedusaGo docs do not use analytics: the provider keeps the upstream
// interface and drops every event.
import React, { createContext, useContext } from "react"

export type ExtraData = {
  section?: string
  [key: string]: unknown
}

type Trackers = "segment" | "posthog"

export type TrackedEvent = {
  event: string
  options?: Record<string, unknown>
  callback?: () => void
  tracker?: Trackers | Trackers[]
}

export type AnalyticsContextType = {
  track: ({
    event,
    instant,
  }: {
    event: TrackedEvent
    instant?: boolean
  }) => void
}

const AnalyticsContext = createContext<AnalyticsContextType | null>(null)

export type AnalyticsProviderProps = {
  segmentWriteKey?: string
  reoDevKey?: string
  children?: React.ReactNode
}

const track: AnalyticsContextType["track"] = ({ event }) => {
  event.callback?.()
}

export const AnalyticsProvider = ({ children }: AnalyticsProviderProps) => (
  <AnalyticsContext.Provider value={{ track }}>
    {children}
  </AnalyticsContext.Provider>
)

export const useAnalytics = () => {
  const context = useContext(AnalyticsContext)
  if (!context) {
    throw new Error("useAnalytics must be used within a AnalyticsProvider")
  }
  return context
}
