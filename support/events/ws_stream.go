package events

import (
	"encoding/json"
	"golang.org/x/net/websocket"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
)

type event struct {
	Name    string      `json:"event"`
	Payload interface{} `json:"payload"`
}

func NewWebsocketStream() *wsStream {
	return &wsStream{}
}

type wsStream struct {
	lock      sync.RWMutex
	listeners []io.WriteCloser
}

func (ws *wsStream) cloneListeners() []io.WriteCloser {
	ws.lock.RLock()
	defer ws.lock.RUnlock()
	n := len(ws.listeners)
	if n == 0 {
		return nil
	}
	cp := make([]io.WriteCloser, n)
	copy(cp, ws.listeners)
	return cp
}

func (ws *wsStream) addListener(writer io.WriteCloser) {
	ws.lock.Lock()
	defer ws.lock.Unlock()
	ws.listeners = append(ws.listeners, writer)
}

func (ws *wsStream) removeListener(writer io.WriteCloser) {
	ws.lock.Lock()
	defer ws.lock.Unlock()
	for i, w := range ws.listeners {
		if w == writer {
			last := len(ws.listeners) - 1
			ws.listeners[last], ws.listeners[i] = ws.listeners[i], ws.listeners[last]
			ws.listeners = ws.listeners[:last]
			break
		}
	}
}

func (ws *wsStream) Feed(eventName string, payload interface{}) {
	listeners := ws.cloneListeners()
	data, _ := json.Marshal(event{
		Name:    eventName,
		Payload: payload,
	})
	for _, listener := range listeners {
		_, err := listener.Write(data)
		if err != nil {
			_ = listener.Close()
		}
	}
}

func (ws *wsStream) Handler() http.Handler {
	return websocket.Handler(ws.serveWS)
}

func (ws *wsStream) Close() {
	for _, lst := range ws.cloneListeners() {
		_ = lst.Close()
	}
}

func (ws *wsStream) serveWS(c *websocket.Conn) {
	ws.addListener(c)
	_, _ = io.Copy(ioutil.Discard, c)
	ws.removeListener(c)
	c.Close()
}
