import { useEffect } from "react"
import "./App.css"
import OptionTable from "./components/OptionTable"
import { initWebSocket } from "./ws/socket"

export default function App() {
  useEffect(() => {
    const ws = initWebSocket()

    return () => {
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
        ws.close()
      }
    }
  }, [])

  return <OptionTable />
}
