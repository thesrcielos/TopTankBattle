package state

import (
	"log"
	"sync"

	"time"

	"github.com/gorilla/websocket"
)

type Position struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Angle float64 `json:"angle"`
}

type Bullet struct {
	ID       string   `json:"id"`
	Position Position `json:"position"`
	Speed    float64  `json:"speed"`
	OwnerId  string   `json:"ownerId"`
}

type Fortress struct {
	ID         string     `json:"id"`
	Position   Position   `json:"position"`
	Health     int        `json:"health"`
	Team1      bool       `json:"team1"`
	FortressMu sync.Mutex `json:"-"`
}

type PlayerState struct {
	ID       string     `json:"id"`
	Position Position   `json:"position"`
	Health   int        `json:"health"`
	Team1    bool       `json:"team1"`
	PlayerMu sync.Mutex `json:"-"`
}

type GameState struct {
	Timestamp  int64                   `json:"timestamp"`
	Players    map[string]*PlayerState `json:"players"`
	Bullets    map[string]*Bullet      `json:"bullets"`
	Fortresses []*Fortress             `json:"fortress"`
	RoomId     string                  `json:"-"`
	GameMu     sync.Mutex              `json:"-"`
}

type PlayerConnection struct {
	ID        string
	RoomId    string
	GameState *GameState
	Connected bool
	Conn      *websocket.Conn
	ConnMu    sync.Mutex
}

var (
	players   = make(map[string]*PlayerConnection)
	playersMu sync.RWMutex
)

func RegisterPlayer(id string, roomId string, conn *websocket.Conn) {
	player := GetPlayer(id)
	playersMu.Lock()
	defer playersMu.Unlock()
	if player == nil {
		players[id] = &PlayerConnection{
			ID:        id,
			Connected: true,
			Conn:      conn,
			GameState: nil,
			RoomId:    roomId,
		}
	} else {
		player.ConnMu.Lock()
		player.Conn = conn
		player.Connected = true
		player.ConnMu.Unlock()
	}
}

func UnregisterPlayerDelayed(id string, delay time.Duration, LeaveRoom func(string) error) {
	go func() {
		playersMu.Lock()
		player := players[id]
		if player == nil {
			playersMu.Unlock()
			return
		}
		player.ConnMu.Lock()
		player.Connected = false
		player.ConnMu.Unlock()
		playersMu.Unlock()

		time.Sleep(delay)

		playersMu.Lock()
		player = players[id]
		if player != nil && !player.Connected {
			delete(players, id)
			playersMu.Unlock()

			LeaveRoom(id)
			log.Printf("Player %s removed for %s seconds of inactivity", id, delay)
		} else {
			playersMu.Unlock()
			log.Printf("Player %s Reconnected on time", id)
		}
	}()
}

func UnregisterPlayer(id string) {
	player := GetPlayer(id)
	if player == nil {
		return
	}
	if player.Conn != nil {
		player.Conn.Close()
	}

	delete(players, id)
}
func GetPlayer(id string) *PlayerConnection {
	playersMu.RLock()
	defer playersMu.RUnlock()

	return players[id]
}

func GetAllPlayers() []*PlayerConnection {
	playersMu.RLock()
	defer playersMu.RUnlock()

	all := make([]*PlayerConnection, 0, len(players))
	for _, p := range players {
		all = append(all, p)
	}
	return all
}
