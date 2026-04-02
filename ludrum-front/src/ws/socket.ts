import { useOptionStore } from "../store/useOptionStore"
import { buildAuthedWsUrl } from "../config"
import type { StreamMessage } from "../types/option"

let socket: WebSocket | null = null

export function initWebSocket() {
  if (
    socket &&
    (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)
  ) {
    return socket
  }

  socket = new WebSocket(buildAuthedWsUrl())

  socket.onopen = () => {
    console.log("WS connected")
  }

  socket.onclose = (event) => {
    if (event.code !== 1006) {
      console.warn("WS closed", event.code)
    }
  }

  socket.onerror = () => {
    console.warn("WS error")
  }

  socket.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data) as StreamMessage
      const store = useOptionStore.getState()

      if (!message?.data) return

      if (message.type === "snapshot" || message.type === "delta") {
        // console.log(`${message.type.toUpperCase()} PAYLOAD:`, message)
        store.hydrate(message.data, message.type)
      }
    } catch (error) {
      console.error("WS parse error", error)
    }
  }

  return socket
}
