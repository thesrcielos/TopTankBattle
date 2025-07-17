package game

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/thesrcielos/TopTankBattle/internal/apperrors"
	"github.com/thesrcielos/TopTankBattle/internal/game/maps"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
	"github.com/thesrcielos/TopTankBattle/websocket/transport"
)

const tileSize = 32
const MAP_HEIGHT = 832
const MAP_WIDTH = 1984

func StartGame(playerId string, roomId string) error {
	room, err := getRoom(roomId)
	if err != nil {
		fmt.Println("Error ", err)
		return err
	}

	if err := validateRoom(room, playerId); err != nil {
		fmt.Println("Error ", err)
		return err
	}

	gameState := &state.GameState{
		Timestamp:  time.Now().Unix(),
		Players:    make(map[string]*state.PlayerState),
		Bullets:    make(map[string]*state.Bullet),
		RoomId:     roomId,
		Fortresses: []*state.Fortress{},
	}

	fortress1 := &state.Fortress{
		ID: "1",
		Position: state.Position{
			X:     48,
			Y:     416,
			Angle: 0,
		},
		Health: 500,
		Team1:  true,
	}

	fortress2 := &state.Fortress{
		ID: "2",
		Position: state.Position{
			X:     1936,
			Y:     416,
			Angle: 0,
		},
		Health: 500,
		Team1:  false,
	}

	gameState.Fortresses = append(gameState.Fortresses, fortress1)
	gameState.Fortresses = append(gameState.Fortresses, fortress2)

	for i, player := range room.Team1 {
		position := state.Position{
			X:     float64(150),
			Y:     float64(324 + i*80),
			Angle: 0,
		}
		gameState.Players[player.ID] = &state.PlayerState{
			ID:       player.ID,
			Position: position,
			Health:   100,
			Team1:    true,
		}

	}

	for i, player := range room.Team2 {
		position := state.Position{
			X:     float64(1834),
			Y:     float64(324 + i*80),
			Angle: math.Pi,
		}
		gameState.Players[player.ID] = &state.PlayerState{
			ID:       player.ID,
			Position: position,
			Health:   100,
			Team1:    false,
		}
	}

	if err := setPlayersGameState(gameState); err != nil {
		fmt.Println("Error ", err)
		return err
	}

	notifyGameStart(gameState)
	room.Status = "PLAYING"
	saveRoom(*room)
	go RunGameLoop(gameState)
	return nil
}

func notifyGameStart(game *state.GameState) {
	message := GameMessage{
		Type:    "GAME_START",
		Payload: game,
		Users:   getGamePlayerIds(game),
	}
	msg, err := json.Marshal(message)
	if err != nil {
		log.Println("Error encoding message:", err)
		return
	}
	PublishToRoom(game.RoomId, string(msg))
}

func setPlayersGameState(gameState *state.GameState) error {
	if gameState == nil {
		return apperrors.NewAppError(400, "Cannot start game: game state is nil", nil)
	}

	for playerId, _ := range gameState.Players {
		player := state.GetPlayer(playerId)
		if player == nil {
			return apperrors.NewAppError(400, "Cannot start game: player does not exist", nil)
		}
	}

	for playerId, playerState := range gameState.Players {
		player := state.GetPlayer(playerId)
		player.ConnMu.Lock()
		defer player.ConnMu.Unlock()

		if player.GameState != nil {
			return apperrors.NewAppError(400, "Cannot start game: player is already in a game", nil)
		}

		player.PlayerState = playerState
		player.GameState = gameState
	}

	return nil
}

func validateRoom(room *Room, playerId string) error {
	if room == nil {
		return apperrors.NewAppError(400, "Cannot start game: room does not exist", nil)
	}

	if room.Host.ID != playerId {
		return apperrors.NewAppError(403, "Only the host can start the game", nil)
	}
	fmt.Println(room.Status)
	if room.Status != "LOBBY" {
		return apperrors.NewAppError(400, "Cannot start game: room is not in LOBBY status", nil)
	}

	if len(room.Team1) == 0 || len(room.Team2) == 0 {
		return apperrors.NewAppError(400, "Cannot start game: not enough players in the room", nil)
	}

	if len(room.Team1) > 4 || len(room.Team2) > 4 {
		return apperrors.NewAppError(400, "Cannot start game: too many players in a team", nil)
	}

	if math.Abs(float64(len(room.Team1)-len(room.Team2))) > 1 {
		return apperrors.NewAppError(400, "Cannot start game: teams must have at most 1 player more than the other team", nil)
	}

	return nil
}

func MovePlayer(playerId string, newPosition state.Position) {
	player := state.GetPlayer(playerId)
	if player == nil {
		log.Println("Player connection not exists")
		return
	}

	player.PlayerState.PlayerMu.Lock()
	if player.PlayerState.Health <= 0 {
		player.PlayerState.PlayerMu.Unlock()
		return
	}

	player.PlayerState.Position = newPosition
	player.PlayerState.PlayerMu.Unlock()

	message := GameMessage{
		Type: "MOVE",
		Payload: MoveMessage{
			PlayerId: playerId,
			Position: newPosition,
		},
		Users: getGamePlayerIds(player.GameState),
	}
	msg, err := json.Marshal(message)
	if err != nil {
		log.Println("Error encoding message:", err)
		return
	}

	PublishToRoom(player.GameState.RoomId, string(msg))
}

func sendGameChangeMessage(game *state.GameState, msg GameMessage) {
	if game == nil {
		log.Println("Game state is nil")
		return
	}

	message, err := json.Marshal(msg)
	if err != nil {
		log.Println("Error encoding message:", err)
		return
	}

	PublishToRoom(game.RoomId, string(message))
}

func ShootBullet(bullet *state.Bullet) {
	player := state.GetPlayer(bullet.OwnerId)
	if player == nil || player.PlayerState == nil {
		return
	}

	player.PlayerState.PlayerMu.Lock()
	if player.PlayerState.Health <= 0 {
		player.PlayerState.PlayerMu.Unlock()
		return
	}
	team1 := player.PlayerState.Team1
	player.PlayerState.PlayerMu.Unlock()

	game := player.GameState
	game.GameMu.Lock()
	game.Bullets[bullet.ID] = bullet
	game.GameMu.Unlock()

	msg := transport.OutgoingMessage{
		Type: "SHOOT",
		Payload: ShootMessage{
			ID:       bullet.ID,
			Position: bullet.Position,
			Team1:    team1,
			OwnerId:  bullet.OwnerId,
		},
	}

	for playerId, _ := range game.Players {
		if playerId == bullet.OwnerId {
			continue
		}
		transport.SendToPlayer(playerId, msg)
	}
}

func RunGameLoop(state *state.GameState) {
	users := getGamePlayerIds(state)
	ticker := time.NewTicker(25 * time.Millisecond) // ~40 FPS
	defer ticker.Stop()
	gameOver := false

	var lastUpdate = time.Now()
	for range ticker.C {
		if gameOver {
			break
		}
		state.GameMu.Lock()
		const bulletDamage = 20

		now := time.Now()
		delta := now.Sub(lastUpdate).Seconds()
		UpdateBullets(state.Bullets, delta)

		for id, bullet := range state.Bullets {
			hitPlayer, hitFortress, hitWall := CheckBulletCollision(bullet, state.Players, state.Fortresses)

			if hitWall {
				delete(state.Bullets, id)
				continue
			}

			if hitPlayer != nil {
				hitPlayer.Health -= bulletDamage
				delete(state.Bullets, id)
				if hitPlayer.Health > 0 {
					sendGameChangeMessage(state, GameMessage{
						Type: "PLAYER_HIT",
						Payload: map[string]interface{}{
							"playerId": hitPlayer.ID,
							"health":   hitPlayer.Health,
						},
						Users: users,
					})
				} else {
					sendGameChangeMessage(state, GameMessage{
						Type: "PLAYER_KILLED",
						Payload: map[string]interface{}{
							"playerId": hitPlayer.ID,
						},
						Users: users,
					})
					go RevivePlayer(hitPlayer.ID, state)
				}
				continue
			}

			if hitFortress != nil {
				hitFortress.Health -= bulletDamage
				if hitFortress.Health <= 0 {
					sendGameChangeMessage(state, GameMessage{
						Type: "GAME_OVER",
						Payload: map[string]interface{}{
							"team1": !hitFortress.Team1,
						},
						Users: users,
					})
					gameOver = true
					FinishGame(state)
					break
				} else {
					sendGameChangeMessage(state, GameMessage{
						Type: "FORTRESS_HIT",
						Payload: map[string]interface{}{
							"team1":  hitFortress.Team1,
							"health": hitFortress.Health,
						},
						Users: users,
					})
				}
				delete(state.Bullets, id)
				continue
			}
		}
		lastUpdate = now
		state.GameMu.Unlock()
	}
}

func CheckBulletCollision(bullet *state.Bullet, players map[string]*state.PlayerState, fortresses []*state.Fortress) (*state.PlayerState, *state.Fortress, bool) {
	const bulletWidth = 12.0
	x := bullet.Position.X
	y := bullet.Position.Y
	col := int(x) / tileSize
	row := int(y) / tileSize
	team1 := players[bullet.OwnerId].Team1
	obstacles := maps.Matrix

	// 2. Check Obstacle collision
	if row >= 0 && row < len(obstacles) && col >= 0 && col < len(obstacles[0]) {
		if obstacles[row][col] {
			return nil, nil, true
		}
	} else {
		return nil, nil, true
	}

	// 3. Player Collision
	for _, p := range players {
		if p.ID == bullet.OwnerId || p.Health <= 0 {
			continue
		}

		collision := rectCollision(bullet.Position, p.Position, 32, 30)
		if !collision {
			continue
		}
		if p.Team1 == team1 {
			return nil, nil, true
		}
		return p, nil, false
	}

	// 4. Fortress Collision
	for _, f := range fortresses {
		collision := rectCollision(bullet.Position, f.Position, 64, 256)
		if !collision {
			continue
		}
		if f.Team1 == team1 {
			return nil, nil, true
		}
		return nil, f, false
	}

	return nil, nil, false
}

func rectCollision(point state.Position, center state.Position, width, height float64) bool {
	halfW := width / 2
	halfH := height / 2

	return point.X >= center.X-halfW &&
		point.X <= center.X+halfW &&
		point.Y >= center.Y-halfH &&
		point.Y <= center.Y+halfH
}

func UpdateBullets(bullets map[string]*state.Bullet, delta float64) {
	for _, b := range bullets {
		b.Position.X += math.Cos(b.Position.Angle) * b.Speed * delta
		b.Position.Y += math.Sin(b.Position.Angle) * b.Speed * delta
	}
}

func RevivePlayer(playerId string, gameState *state.GameState) {
	time.Sleep(6 * time.Second)
	player := gameState.Players[playerId]
	player.PlayerMu.Lock()
	player.Health = 100
	player.PlayerMu.Unlock()
	seed := time.Now().UnixNano()
	source := rand.NewSource(seed)
	r := rand.New(source)
	x := 150
	angle := float64(0)
	if !player.Team1 {
		x = 1834
		angle = math.Pi
	}
	sendGameChangeMessage(gameState, GameMessage{
		Type: "PLAYER_REVIVED",
		Payload: map[string]interface{}{
			"playerId": playerId,
			"position": state.Position{
				X:     float64(x),
				Y:     float64(244 + 80*r.Intn(6)),
				Angle: angle,
			},
		},
		Users: getGamePlayerIds(gameState),
	})
}

func FinishGame(game *state.GameState) {
	for id, _ := range game.Players {
		player := state.GetPlayer(id)
		if player == nil {
			continue
		}
		player.ConnMu.Lock()
		player.GameState = nil
		player.PlayerState = nil
		player.ConnMu.Unlock()
	}
	room, err := getRoom(game.RoomId)
	if err != nil {
		log.Println("error getting ", err)
		return
	}
	room.Status = "LOBBY"
	errDB := saveRoom(*room)
	if errDB != nil {
		log.Println("error saving ", errDB)
		return
	}

	msg := GameMessage{
		Type:    "ROOM_INFO",
		Payload: room,
	}

	sendRoomChangeMessage(room, msg)
}

func getGamePlayerIds(game *state.GameState) []string {
	playerIds := make([]string, 0, len(game.Players))
	for id := range game.Players {
		playerIds = append(playerIds, id)
	}
	return playerIds
}
