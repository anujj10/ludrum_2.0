package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"ludrum/internal/runtime"
	"github.com/gorilla/websocket"
	"ludrum/internal/storage/redis"
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
	done chan struct{}
	once sync.Once
}

func (c *Client) close() {
	c.once.Do(func() {
		close(c.done)
	})
}

type Hub struct {
	clients map[*Client]bool

	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 1024),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Println("Client connected. Total:", len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.close()
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case <-client.done:
					delete(h.clients, client)
				case client.send <- message:
				default:
					select {
					case <-client.send:
					default:
					}

					select {
					case <-client.done:
						delete(h.clients, client)
					case client.send <- message:
					default:
					}
				}
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	writeWait  = 5 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

func startRedisSubscriber(redisClient *redis.RedisClient, hub *Hub, channel string) {
	ctx := context.Background()

	sub := redisClient.Client.Subscribe(ctx, channel)
	defer sub.Close()

	ch := sub.Channel()

	log.Println("Subscribed:", channel)

	for msg := range ch {
		if msg == nil || msg.Payload == "" {
			continue
		}

		parts := strings.Split(channel, ":")
		msgType := parts[len(parts)-1]
		envelope := []byte(`{"type":"` + msgType + `","data":` + msg.Payload + `}`)

		hub.broadcast <- envelope
	}
}

func StartWS(redisClient *redis.RedisClient, port string, runtimeManager *runtime.Manager) {
	hub := NewHub()
	go hub.Run()

	if runtimeManager == nil {
		prefix := redisClient.GetPrefix()
		go startRedisSubscriber(redisClient, hub, prefix+":snapshot")
		go startRedisSubscriber(redisClient, hub, prefix+":delta")
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		user, err := authorizeRequest(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var userRuntime *runtime.UserRuntime
		if runtimeManager != nil {
			userRuntime, err = ensureFyersRuntimeForUser(r.Context(), runtimeManager, user.ID)
			if err != nil {
				http.Error(w, "broker runtime unavailable", http.StatusServiceUnavailable)
				return
			}
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("upgrade error:", err)
			return
		}

		client := &Client{
			conn: conn,
			send: make(chan []byte, 1024),
			done: make(chan struct{}),
		}

		hub.register <- client

		go func() {
			if userRuntime != nil {
				if payload, _, ok := userRuntime.LatestPayload(); ok {
					raw, marshalErr := json.Marshal(map[string]interface{}{
						"type": payload.Type,
						"data": payload,
					})
					if marshalErr == nil {
						select {
						case <-client.done:
							return
						case client.send <- raw:
						}
					}
				}
				return
			}

			val, err := redisClient.GetLatestPayload()
			if err == nil && len(val) > 0 {
				select {
				case <-client.done:
					return
				case client.send <- []byte(`{"type":"snapshot","data":` + string(val) + `}`):
				}
			}
		}()

		if userRuntime != nil {
			go func() {
				ticker := time.NewTicker(time.Second)
				defer ticker.Stop()

				var lastRevision uint64

				for {
					select {
					case <-client.done:
						return
					case <-ticker.C:
						payload, revision, ok := userRuntime.LatestPayload()
						if !ok || revision == lastRevision {
							continue
						}
						lastRevision = revision

						raw, marshalErr := json.Marshal(map[string]interface{}{
							"type": payload.Type,
							"data": payload,
						})
						if marshalErr != nil {
							continue
						}

						select {
						case <-client.done:
							return
						case client.send <- raw:
						}
					}
				}
			}()
		}

		go func() {
			ticker := time.NewTicker(pingPeriod)

			defer func() {
				ticker.Stop()
				hub.unregister <- client
				_ = conn.Close()
			}()

			for {
				select {
				case <-client.done:
					return

				case msg := <-client.send:
					conn.SetWriteDeadline(time.Now().Add(writeWait))
					if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
						return
					}

				case <-ticker.C:
					conn.SetWriteDeadline(time.Now().Add(writeWait))
					if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						return
					}
				}
			}
		}()

		go func() {
			defer func() {
				hub.unregister <- client
				_ = conn.Close()
			}()

			conn.SetReadLimit(512)
			conn.SetReadDeadline(time.Now().Add(pongWait))
			conn.SetPongHandler(func(string) error {
				conn.SetReadDeadline(time.Now().Add(pongWait))
				return nil
			})

			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}()
	})

	log.Println("WS running on :" + port)

	go func() {
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()
}
