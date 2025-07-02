package transport

import (
	"log"

	"github.com/thesrcielos/TopTankBattle/internal/game/state"
)

type OutgoingMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func SendToPlayer(playerID string, msg OutgoingMessage) {
	player := state.GetPlayer(playerID)
	if player == nil {
		return
	}

	player.ConnMu.Lock()
	defer player.ConnMu.Unlock()

	if err := player.Conn.WriteJSON(msg); err != nil {
		log.Println("Error sending msg to", playerID, ":", err)
	}
}

func BroadcastToPlayers(players *[]string, msg OutgoingMessage) {
	if players == nil {
		return
	}

	for _, player := range *players {
		SendToPlayer(player, msg)
	}
}

func BroadcastExcept(excludeId string, players *[]string, msg OutgoingMessage) {
	if players == nil {
		return
	}

	for _, player := range *players {
		if player != excludeId {
			SendToPlayer(player, msg)
		}
	}
}
