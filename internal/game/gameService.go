package game

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/thesrcielos/TopTankBattle/internal/apperrors"
	"github.com/thesrcielos/TopTankBattle/internal/game/maps"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
	"github.com/thesrcielos/TopTankBattle/internal/user"
)

const tileSize = 32
const MAP_HEIGHT = 832
const MAP_WIDTH = 1984

type LeaderElector interface {
	AttemptLeadership(roomId string)
}

type GameService interface {
	StartGame(playerId string, roomId string, test bool) error
	NotifyGameStart(game *state.GameState)
	SetPlayersGameState(gameState *state.GameState) error
	ValidateRoom(room *Room, playerId string) error
	MovePlayer(playerId string, newPosition state.Position)
	SendGameChangeMessage(roomId string, msg GameMessage)
	ShootBullet(bullet *state.Bullet)
	getPlayerIdsFromRoomAndTeam(roomId string, playerId string) ([]string, bool)
	getPlayerIdsFromRoom(roomId string, playerId string) []string
	RunGameLoop(state *state.GameState, test bool)
	HandleHitFortress(hitFortress *state.Fortress, state *state.GameState, bulletDamage int, bulletId string, users []string) bool
	HandleHitPlayer(hitPlayer *state.PlayerState, state *state.GameState, bulletDamage int, bulletId string, users []string)
	CheckBulletCollision(bullet *state.Bullet, players map[string]*state.PlayerState, fortresses []*state.Fortress) (*state.PlayerState, *state.Fortress, bool)
	checkFortressCollision(checkPoints []struct{ x, y float64 }, fortress *state.Fortress, team1 bool) (*state.Fortress, bool)
	checkPlayerCollision(checkPoints []struct{ x, y float64 }, player *state.PlayerState, team1 bool) (*state.PlayerState, bool)
	checkObstacleCollision(point struct{ x, y float64 }, obstacles [][]bool) bool
	rectCollision(point state.Position, center state.Position, width, height float64) bool
	UpdateBullets(bullets map[string]*state.Bullet, delta float64)
	RevivePlayer(playerId string, gameState *state.GameState)
	FinishGame(game *state.GameState)
	getGamePlayerIds(game *state.GameState, playerId string) []string
}

type GameServiceImpl struct {
	roomService *RoomService
	roomRepo    RoomRepository
	userService *user.UserService
	repo        GameStateRepository
}

func NewGameService(repo GameStateRepository, roomRepo RoomRepository, roomService *RoomService, userService *user.UserService) *GameServiceImpl {
	return &GameServiceImpl{repo: repo,
		roomRepo:    roomRepo,
		roomService: roomService,
		userService: userService,
	}
}

func (s *GameServiceImpl) StartGame(playerId string, roomId string, test bool) error {
	room, err := s.roomRepo.GetRoom(roomId)
	if err != nil {
		fmt.Println("Error Obtainig room", err)
		return err
	}

	if err := s.ValidateRoom(room, playerId); err != nil {
		fmt.Println("Error validating room", err)
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

	if err := s.SetPlayersGameState(gameState); err != nil {
		fmt.Println("Error setting game state", err)
		return err
	}

	s.NotifyGameStart(gameState)
	room.Status = "PLAYING"
	s.roomRepo.SaveRoom(room)
	s.repo.SaveGameState(gameState)
	go s.RunGameLoop(gameState, test)
	s.repo.TryToBecomeLeader(roomId)
	msg := GameMessage{
		Type: "GAME_START_INFO",
		Payload: map[string]string{
			"roomId":   roomId,
			"instance": instanceID,
		},
	}
	s.SendGameChangeMessage(roomId, msg)
	return nil
}

func (s *GameServiceImpl) NotifyGameStart(game *state.GameState) {
	message := GameMessage{
		Type:    "GAME_START",
		Payload: game,
		Users:   s.getGamePlayerIds(game, ""),
	}
	msg, err := json.Marshal(message)
	if err != nil {
		log.Println("Error encoding message:", err)
		return
	}
	s.repo.PublishToRoom(string(msg))
}

func (s *GameServiceImpl) SetPlayersGameState(gameState *state.GameState) error {
	if gameState == nil {
		return apperrors.NewAppError(400, "Cannot start game: game state is nil", nil)
	}

	for playerId := range gameState.Players {
		player := state.GetPlayer(playerId)
		if player == nil {
			continue
		}
		player.ConnMu.Lock()
		defer player.ConnMu.Unlock()

		if player.GameState != nil {
			return apperrors.NewAppError(400, "Cannot start game: player is already in a game", nil)
		}

		player.GameState = gameState
	}

	return nil
}

func (s *GameServiceImpl) AttemptLeadership(roomId string) {
	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if s.repo.TryToBecomeLeader(roomId) {
			fmt.Printf("[INFO] Instance %s is now leader of the room %s\n", instanceID, roomId)

			// Recuperar estado anterior desde Redis
			state := s.repo.RestoreGameState(roomId)

			s.RunGameLoop(state, false)
		}
	}
}

func (s *GameServiceImpl) ValidateRoom(room *Room, playerId string) error {
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

func (s *GameServiceImpl) MovePlayer(playerId string, newPosition state.Position) {
	player := state.GetPlayer(playerId)
	if player == nil {
		log.Println("Player connection not exists")
		return
	}

	message := GameMessage{
		Type: "MOVE",
		Payload: MoveMessage{
			PlayerId: playerId,
			Position: newPosition,
		},
	}

	if player.GameState == nil {
		message.Users = s.getPlayerIdsFromRoom(player.RoomId, player.ID)
		s.SendGameChangeMessage(player.RoomId, message)
		gameMessage := GameMessage{
			Type: "GAME_MOVE",
			Payload: MoveMessage{
				PlayerId: playerId,
				Position: newPosition,
			},
		}
		s.SendGameChangeMessage(player.RoomId, gameMessage)
		return
	}

	message.Users = s.getGamePlayerIds(player.GameState, playerId)
	playerState := player.GameState.Players[playerId]
	playerState.PlayerMu.Lock()
	if playerState.Health <= 0 {
		playerState.PlayerMu.Unlock()
		return
	}

	playerState.Position = newPosition
	playerState.PlayerMu.Unlock()
	s.SendGameChangeMessage(player.GameState.RoomId, message)
}

func (s *GameServiceImpl) SendGameChangeMessage(roomId string, msg GameMessage) {
	if roomId == "" {
		log.Println("Game state is nil")
		return
	}

	message, err := json.Marshal(msg)
	if err != nil {
		log.Println("Error encoding message:", err)
		return
	}

	s.repo.PublishToRoom(string(message))
}

func (s *GameServiceImpl) ShootBullet(bullet *state.Bullet) {
	player := state.GetPlayer(bullet.OwnerId)
	if player == nil {
		return
	}

	msg := GameMessage{
		Type: "SHOOT",
	}

	if player.GameState == nil {
		players, team1 := s.getPlayerIdsFromRoomAndTeam(player.RoomId, bullet.OwnerId)
		msg.Payload = ShootMessage{
			ID:       bullet.ID,
			Position: bullet.Position,
			Team1:    team1,
			OwnerId:  bullet.OwnerId,
		}
		msg.Users = players
		s.SendGameChangeMessage(player.RoomId, msg)

		gameMessage := GameMessage{
			Type:    "GAME_SHOOT",
			Payload: bullet,
		}
		s.SendGameChangeMessage(player.RoomId, gameMessage)
		return
	}

	game := player.GameState
	playerState := game.Players[bullet.OwnerId]
	playerState.PlayerMu.Lock()
	if playerState.Health <= 0 {
		playerState.PlayerMu.Unlock()
		return
	}
	team1 := playerState.Team1
	msg.Payload = ShootMessage{
		ID:       bullet.ID,
		Position: bullet.Position,
		Team1:    team1,
		OwnerId:  bullet.OwnerId,
	}

	msg.Users = s.getGamePlayerIds(game, bullet.OwnerId)
	playerState.PlayerMu.Unlock()

	game.GameMu.Lock()
	game.Bullets[bullet.ID] = bullet
	game.GameMu.Unlock()
	s.SendGameChangeMessage(game.RoomId, msg)
}

func (s *GameServiceImpl) getPlayerIdsFromRoomAndTeam(roomId string, playerId string) ([]string, bool) {
	room, err := s.roomRepo.GetRoom(roomId)
	if err != nil {
		return nil, false
	}

	team1 := false
	playerIds := make([]string, 0, len(room.Team1)+len(room.Team2))
	for _, player := range room.Team1 {
		if player.ID == playerId {
			team1 = true
			continue
		}
		playerIds = append(playerIds, player.ID)
	}
	for _, player := range room.Team2 {
		if player.ID == playerId {
			continue
		}
		playerIds = append(playerIds, player.ID)
	}

	return playerIds, team1
}

func (s *GameServiceImpl) getPlayerIdsFromRoom(roomId string, playerId string) []string {
	room, err := s.roomRepo.GetRoom(roomId)
	if err != nil {
		return nil
	}

	playerIds := make([]string, 0, len(room.Team1)+len(room.Team2))
	for _, player := range room.Team1 {
		if player.ID == playerId {
			continue
		}
		playerIds = append(playerIds, player.ID)
	}
	for _, player := range room.Team2 {
		if player.ID == playerId {
			continue
		}
		playerIds = append(playerIds, player.ID)
	}

	return playerIds
}

func (s *GameServiceImpl) RunGameLoop(state *state.GameState, test bool) {
	if test {
		return
	}
	users := s.getGamePlayerIds(state, "")
	ticker := time.NewTicker(25 * time.Millisecond) // ~40 FPS
	defer ticker.Stop()
	gameOver := false

	const fixeDelta = 0.025 // Fixed delta time for physics updates
	for range ticker.C {
		if gameOver {
			break
		}

		state.GameMu.Lock()
		const bulletDamage = 20

		s.UpdateBullets(state.Bullets, fixeDelta)

		for id, bullet := range state.Bullets {
			hitPlayer, hitFortress, hitWall := s.CheckBulletCollision(bullet, state.Players, state.Fortresses)
			if hitWall {
				delete(state.Bullets, id)
				continue
			}

			if hitPlayer != nil {
				s.HandleHitPlayer(hitPlayer, state, bulletDamage, id, users)
				continue
			}

			if hitFortress != nil {
				if s.HandleHitFortress(hitFortress, state, bulletDamage, id, users) {
					gameOver = true
					break
				}
				continue
			}
		}
		s.repo.SaveGameState(state)
		state.GameMu.Unlock()

		renew, err := s.repo.RenewLeadership(state.RoomId, 5000*time.Millisecond)
		if err != nil {
			continue
		}

		if !renew {
			s.AttemptLeadership(state.RoomId)
			return
		}
	}
}

func (s *GameServiceImpl) HandleHitFortress(hitFortress *state.Fortress, state *state.GameState, bulletDamage int, bulletId string, users []string) bool {
	hitFortress.Health -= bulletDamage
	if hitFortress.Health <= 0 {
		s.SendGameChangeMessage(state.RoomId, GameMessage{
			Type: "GAME_OVER",
			Payload: map[string]interface{}{
				"team1": !hitFortress.Team1,
			},
			Users: users,
		})
		s.FinishGame(state)
		return true
	} else {
		s.SendGameChangeMessage(state.RoomId, GameMessage{
			Type: "FORTRESS_HIT",
			Payload: map[string]interface{}{
				"team1":  hitFortress.Team1,
				"health": hitFortress.Health,
			},
			Users: users,
		})
		delete(state.Bullets, bulletId)
		return false
	}
}

func (s *GameServiceImpl) HandleHitPlayer(hitPlayer *state.PlayerState, state *state.GameState, bulletDamage int, bulletId string, users []string) {
	hitPlayer.Health -= bulletDamage
	delete(state.Bullets, bulletId)
	if hitPlayer.Health > 0 {
		s.SendGameChangeMessage(state.RoomId, GameMessage{
			Type: "PLAYER_HIT",
			Payload: map[string]interface{}{
				"playerId": hitPlayer.ID,
				"health":   hitPlayer.Health,
			},
			Users: users,
		})
	} else {
		s.SendGameChangeMessage(state.RoomId, GameMessage{
			Type: "PLAYER_KILLED",
			Payload: map[string]interface{}{
				"playerId": hitPlayer.ID,
			},
			Users: users,
		})
		go s.RevivePlayer(hitPlayer.ID, state)
	}
}

func (s *GameServiceImpl) CheckBulletCollision(bullet *state.Bullet, players map[string]*state.PlayerState, fortresses []*state.Fortress) (*state.PlayerState, *state.Fortress, bool) {
	const bulletWidth = 12.0
	const halfWidth = bulletWidth / 2.0
	angle := bullet.Position.Angle

	perpX := -math.Sin(angle)
	perpY := math.Cos(angle)

	centerX := bullet.Position.X
	centerY := bullet.Position.Y

	leftX := centerX + perpX*halfWidth
	leftY := centerY + perpY*halfWidth

	rightX := centerX - perpX*halfWidth
	rightY := centerY - perpY*halfWidth

	checkPoints := []struct{ x, y float64 }{
		{centerX, centerY},
		{leftX, leftY},
		{rightX, rightY},
	}

	team1 := players[bullet.OwnerId].Team1
	obstacles := maps.Matrix

	// 1. Check Obstacle collision para cada punto
	for _, point := range checkPoints {
		col := s.checkObstacleCollision(point, obstacles)
		if col {
			return nil, nil, true
		}
	}

	// 2. Player Collision
	for _, p := range players {
		if p.ID == bullet.OwnerId || p.Health <= 0 {
			continue
		}
		player, hasCollision := s.checkPlayerCollision(checkPoints, p, team1)
		if player != nil {
			return player, nil, false
		}
		if hasCollision {
			return nil, nil, true
		}
	}

	// 3. Fortress Collision
	for _, f := range fortresses {
		fortress, hasCollision := s.checkFortressCollision(checkPoints, f, team1)
		if fortress != nil {
			return nil, fortress, false
		}
		if hasCollision {
			return nil, nil, true
		}
	}

	return nil, nil, false
}

func (s *GameServiceImpl) checkFortressCollision(checkPoints []struct{ x, y float64 }, fortress *state.Fortress, team1 bool) (*state.Fortress, bool) {
	hasCollision := false
	for _, point := range checkPoints {
		bulletPos := state.Position{X: point.x, Y: point.y}
		collision := s.rectCollision(bulletPos, fortress.Position, 64, 256)
		if collision {
			hasCollision = true
			break
		}
	}

	if !hasCollision {
		return nil, false
	}

	if fortress.Team1 == team1 {
		return nil, true
	}
	return fortress, false
}

func (s *GameServiceImpl) checkPlayerCollision(checkPoints []struct{ x, y float64 }, player *state.PlayerState, team1 bool) (*state.PlayerState, bool) {
	hasCollision := false
	for _, point := range checkPoints {
		bulletPos := state.Position{X: point.x, Y: point.y}
		collision := s.rectCollision(bulletPos, player.Position, 32, 30)
		if collision {
			hasCollision = true
			break
		}
	}

	if !hasCollision {
		return nil, false
	}

	if player.Team1 == team1 {
		return nil, true
	}
	return player, false
}

func (s *GameServiceImpl) checkObstacleCollision(point struct{ x, y float64 }, obstacles [][]bool) bool {
	col := int(point.x) / tileSize
	row := int(point.y) / tileSize

	if row >= 0 && row < len(obstacles) && col >= 0 && col < len(obstacles[0]) {
		if obstacles[row][col] {
			return true
		}
	} else {
		return true
	}
	return false
}

func (s *GameServiceImpl) rectCollision(point state.Position, center state.Position, width, height float64) bool {
	halfW := width / 2
	halfH := height / 2

	return point.X >= center.X-halfW &&
		point.X <= center.X+halfW &&
		point.Y >= center.Y-halfH &&
		point.Y <= center.Y+halfH
}

func (s *GameServiceImpl) UpdateBullets(bullets map[string]*state.Bullet, delta float64) {
	for _, b := range bullets {
		b.Position.X += math.Cos(b.Position.Angle) * b.Speed * delta
		b.Position.Y += math.Sin(b.Position.Angle) * b.Speed * delta
	}
}

func (s *GameServiceImpl) RevivePlayer(playerId string, gameState *state.GameState) {
	time.Sleep(6 * time.Second)
	player := gameState.Players[playerId]
	player.PlayerMu.Lock()
	player.Health = 100
	seed := time.Now().UnixNano()
	source := rand.NewSource(seed)
	r := rand.New(source)
	x := 150
	angle := float64(0)
	if !player.Team1 {
		x = 1834
		angle = math.Pi
	}
	pos := state.Position{
		X:     float64(x),
		Y:     float64(244 + 80*r.Intn(6)),
		Angle: angle,
	}
	player.Position = pos
	player.PlayerMu.Unlock()
	s.SendGameChangeMessage(gameState.RoomId, GameMessage{
		Type: "PLAYER_REVIVED",
		Payload: map[string]interface{}{
			"playerId": playerId,
			"position": pos,
		},
		Users: s.getGamePlayerIds(gameState, ""),
	})
}

func (s *GameServiceImpl) FinishGame(game *state.GameState) {
	team2Wins := true
	if game.Fortresses[0].Team1 {
		team2Wins = game.Fortresses[0].Health <= 0
	} else {
		team2Wins = game.Fortresses[1].Health <= 0
	}

	for id, _ := range game.Players {
		player := state.GetPlayer(id)
		if player == nil {
			continue
		}
		player.ConnMu.Lock()
		player.GameState = nil
		player.ConnMu.Unlock()
	}
	room, err := s.roomRepo.GetRoom(game.RoomId)
	if err != nil {
		log.Println("error getting ", err)
		return
	}
	room.Status = "LOBBY"
	errDB := s.roomRepo.SaveRoom(room)
	if errDB != nil {
		log.Println("error saving ", errDB)
		return
	}

	msg := GameMessage{
		Type:    "ROOM_INFO",
		Payload: room,
	}

	s.roomService.sendRoomChangeMessage(room, msg)
	for id, player := range game.Players {
		userId, err := strconv.Atoi(id)
		if err != nil {
			log.Println("Error converting player ID to int:", err)
			continue
		}
		s.userService.UpdatePlayerStats(userId, team2Wins == !player.Team1)
	}

}

func (s *GameServiceImpl) getGamePlayerIds(game *state.GameState, playerId string) []string {
	if game == nil || game.Players == nil {
		return nil
	}
	playerIds := make([]string, 0, len(game.Players))
	for id := range game.Players {
		if playerId == id {
			continue
		}
		playerIds = append(playerIds, id)
	}
	return playerIds
}
