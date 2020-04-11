package events

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

type event struct {
	Name    string      `json:"event"`
	Payload interface{} `json:"payload"`
}
type wsStreamer struct {
	feedLock sync.Mutex
	writers  []*websocket.Conn
}

func NewWebsocketStream() *wsStreamer {
	return &wsStreamer{}
}

func (ws *wsStreamer) Feed(eventName string, payload interface{}) {
	data, _ := json.Marshal(event{
		Name:    eventName,
		Payload: payload,
	})
	var wg sync.WaitGroup
	ws.feedLock.Lock()
	for _, conn := range ws.writers {
		wg.Add(1)
		go func(conn *websocket.Conn) {
			defer wg.Done()
			err := conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				_ = conn.Close()
			}
		}(conn)
	}
	ws.feedLock.Unlock()
	wg.Wait()
}

func (ws *wsStreamer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  8192,
		WriteBufferSize: 8192,
	}
	defer request.Body.Close()

	conn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ws.feedLock.Lock()
	ws.writers = append(ws.writers, conn)
	ws.feedLock.Unlock()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	ws.feedLock.Lock()
	for i, w := range ws.writers {
		if w == conn {
			N := len(ws.writers)
			ws.writers[i] = ws.writers[N-1]
			ws.writers = ws.writers[:N-1]
			break
		}
	}
	ws.feedLock.Unlock()
}
