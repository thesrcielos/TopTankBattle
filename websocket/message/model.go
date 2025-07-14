package message

import (
	"encoding/json"
)

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type MovePayload struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Angle float64 `json:"angle"`
}

type ShootPayload struct {
	Angle float64 `json:"angle"`
}

type RoomDeletionRequestPayload struct {
	Room string `json:"room"`
}

type RoomDelete struct {
	RoomId string `json:"roomId"`
}

type RoomPlayerLeave struct {
	Player string `json:"player"`
	Room   string `json:"room"`
}

type RoomKickRequestPayload struct {
	PlayerId string `json:"playerId"`
	RoomId   string `json:"roomId"`
}

type GameStartPayload struct {
	RoomId string `json:"roomId"`
}

type GameMovePayload struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Angle float64 `json:"angle"`
}
