const defaultApiBaseUrl = "http://localhost:8080"
const defaultWsUrl = "ws://localhost:8081/ws"
export const AUTH_TOKEN_STORAGE_KEY = "index-options-auth-token"
export const ADMIN_SESSION_STORAGE_KEY = "index-options-admin-session"

function trimTrailingSlash(value: string) {
  return value.replace(/\/+$/, "")
}

export const API_BASE_URL = trimTrailingSlash(import.meta.env.VITE_API_BASE_URL || defaultApiBaseUrl)
export const WS_URL = import.meta.env.VITE_WS_URL || defaultWsUrl

export function authHeaders() {
  const token = window.localStorage.getItem(AUTH_TOKEN_STORAGE_KEY)
  const headers: Record<string, string> = {}
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  return headers
}

export function buildAuthedWsUrl() {
  const token = window.localStorage.getItem(AUTH_TOKEN_STORAGE_KEY)
  if (!token) return WS_URL

  const separator = WS_URL.includes("?") ? "&" : "?"
  return `${WS_URL}${separator}token=${encodeURIComponent(token)}`
}
