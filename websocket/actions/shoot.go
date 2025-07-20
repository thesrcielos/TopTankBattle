package actions

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/thesrcielos/TopTankBattle/internal/game"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
	"github.com/thesrcielos/TopTankBattle/websocket/message"
)

func HandleShoot(playerId string, msg message.Message, game *game.GameService) {
	var payload struct {
		OwnerId string  `json:"ownerId"`
		X       float64 `json:"x"`
		Y       float64 `json:"y"`
		Angle   float64 `json:"angle"`
	}

	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return
	}

	bullet := &state.Bullet{
		ID:      uuid.NewString(),
		OwnerId: payload.OwnerId,
		Position: state.Position{
			X:     payload.X,
			Y:     payload.Y,
			Angle: payload.Angle,
		},
		Speed: 500,
	}
	game.ShootBullet(bullet)

}
