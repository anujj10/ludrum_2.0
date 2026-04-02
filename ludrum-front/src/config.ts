const defaultApiBaseUrl = "http://localhost:8080"
const defaultWsUrl = "ws://localhost:8081/ws"

function trimTrailingSlash(value: string) {
  return value.replace(/\/+$/, "")
}

export const API_BASE_URL = trimTrailingSlash(import.meta.env.VITE_API_BASE_URL || defaultApiBaseUrl)
export const WS_URL = import.meta.env.VITE_WS_URL || defaultWsUrl
