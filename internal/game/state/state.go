package state

import (
	"sync"

	"github.com/gorilla/websocket"
)

type PlayerPosition struct {
	X float64
	Y float64
}
type PlayerState struct {
	ID             string
	PlayerPosition *PlayerPosition
	Conn           *websocket.Conn
	ConnMu         sync.Mutex
}

var (
	players   = make(map[string]*PlayerState)
	playersMu sync.RWMutex
)

func RegisterPlayer(id string, conn *websocket.Conn) {
	playersMu.Lock()
	defer playersMu.Unlock()

	players[id] = &PlayerState{
		ID:   id,
		Conn: conn,
	}
}

func UnregisterPlayer(id string) {
	playersMu.Lock()
	defer playersMu.Unlock()

	delete(players, id)
}

func GetPlayer(id string) *PlayerState {
	playersMu.RLock()
	defer playersMu.RUnlock()

	return players[id]
}

func GetAllPlayers() []*PlayerState {
	playersMu.RLock()
	defer playersMu.RUnlock()

	all := make([]*PlayerState, 0, len(players))
	for _, p := range players {
		all = append(all, p)
	}
	return all
}
