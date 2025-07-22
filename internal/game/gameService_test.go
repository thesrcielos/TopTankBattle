package game

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thesrcielos/TopTankBattle/internal/game/maps"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
	"github.com/thesrcielos/TopTankBattle/internal/user"
)

var (
	mockRoomRepo *MockRoomRepository
	mockGameRepo *MockGameStateRepository
	mockUserRepo *user.MockUserRepository

	roomService *RoomService
	userService *user.UserService
)

func TestMain(m *testing.M) {
	// Mock Repositories
	mockRoomRepo = new(MockRoomRepository)
	mockGameRepo = new(MockGameStateRepository)
	mockUserRepo = new(user.MockUserRepository)

	roomService = NewRoomService(mockRoomRepo)
	userService = user.NewUserService(mockUserRepo)

	code := m.Run()
	os.Exit(code)
}

func TestValidateRoomOk(t *testing.T) {
	gameService := NewGameService(mockGameRepo, mockRoomRepo, roomService, userService)
	room := &Room{
		Host:   Player{ID: "host"},
		Status: "LOBBY",
		Team1:  []Player{{ID: "p1"}},
		Team2:  []Player{{ID: "p2"}},
	}
	err := gameService.ValidateRoom(room, "host")
	require.NoError(t, err)
}

var dummyObstacles = [][]bool{
	{false, false, false, false},
	{false, false, false, false},
	{false, false, false, false},
	{false, false, false, false},
}

func TestCheckBulletCollisionHitsPlayer(t *testing.T) {
	gameService := NewGameService(mockGameRepo, mockRoomRepo, roomService, userService)
	// Given
	player := &state.PlayerState{
		ID:       "target",
		Health:   100,
		Team1:    false,
		Position: state.Position{X: 100, Y: 100},
	}
	bullet := &state.Bullet{
		OwnerId: "shooter",
		Position: state.Position{
			X:     100,
			Y:     100,
			Angle: 0,
		},
	}
	shooter := &state.PlayerState{
		ID:    "shooter",
		Team1: true,
	}

	players := map[string]*state.PlayerState{
		"shooter": shooter,
		"target":  player,
	}

	fortresses := []*state.Fortress{}
	maps.Matrix = dummyObstacles

	// When
	pHit, fHit, destroyed := gameService.CheckBulletCollision(bullet, players, fortresses)

	//Then
	if pHit == nil || pHit.ID != "target" {
		t.Errorf("Expected player hit to be 'target', got %v", pHit)
	}
	if fHit != nil {
		t.Errorf("Expected no fortress hit")
	}
	if destroyed {
		t.Errorf("Expected bullet not destroyed")
	}
}

func TestCheckBulletCollisionHitsObstacle(t *testing.T) {
	gameService := NewGameService(mockGameRepo, mockRoomRepo, roomService, userService)
	obstacles := [][]bool{
		{false, false, false},
		{false, true, false}, // [1][1] es obstáculo
		{false, false, false},
	}
	maps.Matrix = obstacles

	bullet := &state.Bullet{
		OwnerId: "p1",
		Position: state.Position{
			X:     float64(32 + 32/2),
			Y:     float64(32 + 32/2),
			Angle: 0,
		},
	}
	players := map[string]*state.PlayerState{
		"p1": {ID: "p1", Team1: true},
	}

	p, f, destroyed := gameService.CheckBulletCollision(bullet, players, nil)
	if !destroyed {
		t.Errorf("Expected bullet to be destroyed by obstacle")
	}
	if p != nil || f != nil {
		t.Errorf("Expected no player or fortress hit")
	}
}

func TestCheckBulletCollisionHitsAllyPlayer(t *testing.T) {
	gameService := NewGameService(mockGameRepo, mockRoomRepo, roomService, userService)
	bullet := &state.Bullet{
		OwnerId: "p1",
		Position: state.Position{
			X:     50,
			Y:     50,
			Angle: 0,
		},
	}
	players := map[string]*state.PlayerState{
		"p1": {ID: "p1", Team1: true},
		"p2": {
			ID:       "p2",
			Team1:    true, // aliado
			Health:   100,
			Position: state.Position{X: 50, Y: 50},
		},
	}
	maps.Matrix = dummyObstacles

	p, f, destroyed := gameService.CheckBulletCollision(bullet, players, nil)
	if destroyed == false {
		t.Errorf("Expected bullet to be destroyed by ally collision")
	}
	if p != nil || f != nil {
		t.Errorf("Expected no kill, only destruction")
	}
}

func TestCheckBulletCollisionHitsEnemyFortress(t *testing.T) {
	gameService := NewGameService(mockGameRepo, mockRoomRepo, roomService, userService)
	bullet := &state.Bullet{
		OwnerId: "p1",
		Position: state.Position{
			X:     50,
			Y:     50,
			Angle: 0,
		},
	}
	players := map[string]*state.PlayerState{
		"p1": {ID: "p1", Team1: true},
	}
	fortresses := []*state.Fortress{
		{
			ID:       "f1",
			Team1:    false,
			Position: state.Position{X: 50, Y: 50},
		},
	}
	maps.Matrix = dummyObstacles

	p, f, destroyed := gameService.CheckBulletCollision(bullet, players, fortresses)
	if f == nil || f.ID != "f1" {
		t.Errorf("Expected enemy fortress hit")
	}
	if p != nil {
		t.Errorf("Expected no player hit")
	}
	if destroyed {
		t.Errorf("Expected bullet not destroyed")
	}
}

func TestStartGameSuccess(t *testing.T) {
	gameService := NewGameService(mockGameRepo, mockRoomRepo, roomService, userService)
	// Datos simulados
	playerID := "player1"
	roomID := "room123"

	room := &Room{
		ID:     roomID,
		Status: "LOBBY",
		Host:   Player{ID: playerID},
		Team1:  []Player{{ID: playerID}},
		Team2:  []Player{{ID: "player2"}},
	}

	// Mock de GetRoom
	mockRoomRepo.On("GetRoom", roomID).Return(room, nil)
	mockRoomRepo.On("SaveRoom", mock.Anything).Return(nil)
	mockGameRepo.On("SaveGameState", mock.Anything).Return()
	mockGameRepo.On("PublishToRoom", mock.Anything).Return()
	mockGameRepo.On("TryToBecomeLeader", roomID).Return(true)

	// Simulación básica del estado global del jugador
	state.RegisterPlayer(playerID, roomID, nil)
	state.RegisterPlayer("player2", roomID, nil)

	err := gameService.StartGame(playerID, roomID, true)

	// Verificar
	assert.NoError(t, err)
	mockRoomRepo.AssertCalled(t, "GetRoom", roomID)
	mockRoomRepo.AssertCalled(t, "SaveRoom", mock.Anything)
	mockGameRepo.AssertCalled(t, "SaveGameState", mock.Anything)
	mockGameRepo.AssertCalled(t, "PublishToRoom", mock.Anything)
	mockGameRepo.AssertCalled(t, "TryToBecomeLeader", roomID)
}

func TestHandleHitPlayerPlayerHitAndKilled(t *testing.T) {
	gameService := NewGameService(mockGameRepo, mockRoomRepo, roomService, userService)
	gameState := &state.GameState{
		RoomId:  "room1",
		Players: map[string]*state.PlayerState{"p1": {ID: "p1", Health: 20}},
		Bullets: map[string]*state.Bullet{"b1": {}},
	}
	users := []string{"p2"}
	mockGameRepo.On("PublishToRoom", mock.Anything).Return()

	gameService.HandleHitPlayer(gameState.Players["p1"], gameState, 20, "b1", users)
	assert.Equal(t, 0, gameState.Players["p1"].Health)
	_, exists := gameState.Bullets["b1"]
	assert.False(t, exists)
	mockGameRepo.AssertCalled(t, "PublishToRoom", mock.Anything)
}

func TestHandleHitPlayerPlayerHitButNotKilled(t *testing.T) {
	gameService := NewGameService(mockGameRepo, mockRoomRepo, roomService, userService)
	gameState := &state.GameState{
		RoomId:  "room1",
		Players: map[string]*state.PlayerState{"p1": {ID: "p1", Health: 100}},
		Bullets: map[string]*state.Bullet{"b1": {}},
	}
	users := []string{"p2"}
	mockGameRepo.On("PublishToRoom", mock.Anything).Return()
	gameService.HandleHitPlayer(gameState.Players["p1"], gameState, 20, "b1", users)
	assert.Equal(t, 80, gameState.Players["p1"].Health)
	_, exists := gameState.Bullets["b1"]
	assert.False(t, exists)
	mockGameRepo.AssertCalled(t, "PublishToRoom", mock.Anything)
}

func TestHandleHitFortressDestroyed(t *testing.T) {
	fortress := &state.Fortress{ID: "f1", Health: 20, Team1: true}
	gameState := &state.GameState{RoomId: "room1", Bullets: map[string]*state.Bullet{"b1": {}}, Fortresses: []*state.Fortress{fortress}}
	users := []string{"p1", "p2"}
	mockGameRepo.On("PublishToRoom", mock.Anything).Return()
	mockGameRepo.On("SaveGameState", mock.Anything).Return()
	mockRoomRepo.On("GetRoom", "room1").Return(&Room{ID: "room1", Host: Player{ID: "host"}}, nil)
	mockRoomRepo.On("SaveRoom", mock.Anything).Return(nil)
	mockRoomRepo.On("PublishToRoom", mock.Anything).Return()
	mockUserRepo.On("GetUserUsername", mock.Anything).Return("user", nil)
	userService = user.NewUserService(mockUserRepo)
	roomService = NewRoomService(mockRoomRepo)
	gameService := NewGameService(mockGameRepo, mockRoomRepo, roomService, userService)
	result := gameService.HandleHitFortress(fortress, gameState, 20, "b1", users)
	assert.True(t, result)
	assert.Equal(t, 0, fortress.Health)
	mockGameRepo.AssertCalled(t, "PublishToRoom", mock.Anything)
}

func TestHandleHitFortressNotDestroyed(t *testing.T) {
	gameService := NewGameService(mockGameRepo, mockRoomRepo, roomService, userService)
	fortress := &state.Fortress{ID: "f1", Health: 100, Team1: true}
	gameState := &state.GameState{RoomId: "room1", Bullets: map[string]*state.Bullet{"b1": {}}, Fortresses: []*state.Fortress{fortress}}
	users := []string{"p1", "p2"}
	mockGameRepo.On("PublishToRoom", mock.Anything).Return()
	result := gameService.HandleHitFortress(fortress, gameState, 20, "b1", users)
	assert.False(t, result)
	assert.Equal(t, 80, fortress.Health)
	_, exists := gameState.Bullets["b1"]
	assert.False(t, exists)
	mockGameRepo.AssertCalled(t, "PublishToRoom", mock.Anything)
}

func TestGetPlayerIdsFromRoom(t *testing.T) {
	localMockRoomRepo := new(MockRoomRepository)
	localGameService := NewGameService(mockGameRepo, localMockRoomRepo, roomService, userService)
	room := &Room{
		ID:    "room1",
		Team1: []Player{{ID: "p1"}, {ID: "p2"}},
		Team2: []Player{{ID: "p3"}},
	}
	localMockRoomRepo.On("GetRoom", "room1").Return(room, nil)
	ids := localGameService.getPlayerIdsFromRoom("room1", "p1")
	assert.ElementsMatch(t, []string{"p2", "p3"}, ids)
}

func TestGetPlayerIdsFromRoomAndTeam(t *testing.T) {
	localMockRoomRepo := new(MockRoomRepository)
	localGameService := NewGameService(mockGameRepo, localMockRoomRepo, roomService, userService)
	room := &Room{
		ID:    "room1",
		Team1: []Player{{ID: "p1"}, {ID: "p2"}},
		Team2: []Player{{ID: "p3"}},
	}
	localMockRoomRepo.On("GetRoom", "room1").Return(room, nil)
	ids, team1 := localGameService.getPlayerIdsFromRoomAndTeam("room1", "p1")
	assert.ElementsMatch(t, []string{"p2", "p3"}, ids)
	assert.True(t, team1)
	ids, team1 = localGameService.getPlayerIdsFromRoomAndTeam("room1", "p3")
	assert.ElementsMatch(t, []string{"p1", "p2"}, ids)
	assert.False(t, team1)
}

func TestMovePlayerSendsMessageWhenPlayerExistsAndAlive(t *testing.T) {
	localMockGameRepo := new(MockGameStateRepository)
	localMockRoomRepo := new(MockRoomRepository)
	localRoomService := NewRoomService(localMockRoomRepo)
	localUserService := user.NewUserService(mockUserRepo)
	gameService := NewGameService(localMockGameRepo, localMockRoomRepo, localRoomService, localUserService)

	playerId := "p1"
	roomId := "room1"
	pos := state.Position{X: 10, Y: 20, Angle: 0}
	state.RegisterPlayer(playerId, roomId, nil)
	playerConn := state.GetPlayer(playerId)
	playerConn.GameState = nil // Simula que no está en partida

	localMockRoomRepo.On("GetRoom", roomId).Return(&Room{
		ID:    roomId,
		Team1: []Player{{ID: playerId}, {ID: "p2"}},
		Team2: []Player{{ID: "p3"}},
	}, nil)
	localMockGameRepo.On("PublishToRoom", mock.Anything).Return()

	gameService.MovePlayer(playerId, pos)
	localMockGameRepo.AssertCalled(t, "PublishToRoom", mock.Anything)
}

func TestMovePlayerNoSendWhenPlayerDoesNotExist(t *testing.T) {
	localMockGameRepo := new(MockGameStateRepository)
	localMockRoomRepo := new(MockRoomRepository)
	localRoomService := NewRoomService(localMockRoomRepo)
	localUserService := user.NewUserService(mockUserRepo)
	gameService := NewGameService(localMockGameRepo, localMockRoomRepo, localRoomService, localUserService)

	// No se registra el jugador
	pos := state.Position{X: 10, Y: 20, Angle: 0}
	// No debe hacer panic ni enviar mensaje
	gameService.MovePlayer("noexiste", pos)
	localMockGameRepo.AssertNotCalled(t, "PublishToRoom", mock.Anything)
}

func TestShootBulletSendsMessageWithoutGameState(t *testing.T) {
	localMockGameRepo := new(MockGameStateRepository)
	localMockRoomRepo := new(MockRoomRepository)
	localRoomService := NewRoomService(localMockRoomRepo)
	localUserService := user.NewUserService(mockUserRepo)
	gameService := NewGameService(localMockGameRepo, localMockRoomRepo, localRoomService, localUserService)

	playerId := "p1"
	roomId := "room1"
	state.RegisterPlayer(playerId, roomId, nil)
	playerConn := state.GetPlayer(playerId)
	playerConn.GameState = nil // Simula que no está en partida

	localMockRoomRepo.On("GetRoom", roomId).Return(&Room{
		ID:    roomId,
		Team1: []Player{{ID: playerId}, {ID: "p2"}},
		Team2: []Player{{ID: "p3"}},
	}, nil)
	localMockGameRepo.On("PublishToRoom", mock.Anything).Return()

	bullet := &state.Bullet{ID: "b1", OwnerId: playerId, Position: state.Position{X: 1, Y: 2, Angle: 0}, Speed: 10}
	gameService.ShootBullet(bullet)
	localMockGameRepo.AssertCalled(t, "PublishToRoom", mock.Anything)
}

func TestShootBulletSendsMessageWithGameState(t *testing.T) {
	localMockGameRepo := new(MockGameStateRepository)
	localMockRoomRepo := new(MockRoomRepository)
	localRoomService := NewRoomService(localMockRoomRepo)
	localUserService := user.NewUserService(mockUserRepo)
	gameService := NewGameService(localMockGameRepo, localMockRoomRepo, localRoomService, localUserService)

	playerId := "p1"
	roomId := "room1"
	state.RegisterPlayer(playerId, roomId, nil)
	playerConn := state.GetPlayer(playerId)
	// Simula que está en partida
	gs := &state.GameState{
		RoomId:  roomId,
		Players: map[string]*state.PlayerState{playerId: {ID: playerId, Health: 100, Team1: true}},
		Bullets: map[string]*state.Bullet{},
	}
	playerConn.GameState = gs

	localMockGameRepo.On("PublishToRoom", mock.Anything).Return()

	bullet := &state.Bullet{ID: "b1", OwnerId: playerId, Position: state.Position{X: 1, Y: 2, Angle: 0}, Speed: 10}
	gameService.ShootBullet(bullet)
	localMockGameRepo.AssertCalled(t, "PublishToRoom", mock.Anything)
}

func TestMovePlayerWithGameStatePlayerAlive(t *testing.T) {
	localMockGameRepo := new(MockGameStateRepository)
	localMockRoomRepo := new(MockRoomRepository)
	localRoomService := NewRoomService(localMockRoomRepo)
	localUserService := user.NewUserService(mockUserRepo)
	gameService := NewGameService(localMockGameRepo, localMockRoomRepo, localRoomService, localUserService)

	playerId := "p1"
	roomId := "room1"
	pos := state.Position{X: 10, Y: 20, Angle: 0}
	state.RegisterPlayer(playerId, roomId, nil)
	playerConn := state.GetPlayer(playerId)
	// Simula que está en partida y vivo
	gs := &state.GameState{
		RoomId:  roomId,
		Players: map[string]*state.PlayerState{playerId: {ID: playerId, Health: 100, Team1: true}},
		Bullets: map[string]*state.Bullet{},
	}
	playerConn.GameState = gs
	localMockGameRepo.On("PublishToRoom", mock.Anything).Return()

	gameService.MovePlayer(playerId, pos)
	localMockGameRepo.AssertCalled(t, "PublishToRoom", mock.Anything)
}

func TestValidateRoomErrors(t *testing.T) {
	gameService := NewGameService(mockGameRepo, mockRoomRepo, roomService, userService)

	room := &Room{Host: Player{ID: "host"}, Status: "LOBBY", Team1: []Player{{ID: "p1"}}, Team2: []Player{{ID: "p2"}}}
	err := gameService.ValidateRoom(nil, "host")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "room does not exist")

	err = gameService.ValidateRoom(room, "notHost")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Only the host can start the game")

	room.Status = "PLAYING"
	err = gameService.ValidateRoom(room, "host")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "room is not in LOBBY status")

	room.Status = "LOBBY"
	room.Team1 = []Player{}
	room.Team2 = []Player{}
	err = gameService.ValidateRoom(room, "host")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enough players")

	room.Team1 = []Player{{ID: "p1"}, {ID: "p2"}, {ID: "p3"}, {ID: "p4"}, {ID: "p5"}}
	room.Team2 = []Player{{ID: "p6"}}
	err = gameService.ValidateRoom(room, "host")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many players")

	room.Team1 = []Player{{ID: "p1"}, {ID: "p2"}, {ID: "p3"}}
	room.Team2 = []Player{{ID: "p4"}}
	err = gameService.ValidateRoom(room, "host")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "teams must have at most 1 player more than the other team")
}

func TestSendGameChangeMessageEmptyRoomId(t *testing.T) {
	localMockGameRepo := new(MockGameStateRepository)
	localMockRoomRepo := new(MockRoomRepository)
	localRoomService := NewRoomService(localMockRoomRepo)
	localUserService := user.NewUserService(mockUserRepo)
	gameService := NewGameService(localMockGameRepo, localMockRoomRepo, localRoomService, localUserService)

	gameService.SendGameChangeMessage("", GameMessage{Type: "TEST", Payload: nil})
	localMockGameRepo.AssertNotCalled(t, "PublishToRoom", mock.Anything)
}

func TestSendGameChangeMessageInvalidJSON(t *testing.T) {
	localMockGameRepo := new(MockGameStateRepository)
	localMockRoomRepo := new(MockRoomRepository)
	localRoomService := NewRoomService(localMockRoomRepo)
	localUserService := user.NewUserService(mockUserRepo)
	gameService := NewGameService(localMockGameRepo, localMockRoomRepo, localRoomService, localUserService)

	msg := GameMessage{Type: "TEST", Payload: make(chan int)}

	gameService.SendGameChangeMessage("room1", msg)
	localMockGameRepo.AssertNotCalled(t, "PublishToRoom", mock.Anything)
}

func TestStartGameFailsOnGetRoom(t *testing.T) {
	localMockGameRepo := new(MockGameStateRepository)
	localMockRoomRepo := new(MockRoomRepository)
	localRoomService := NewRoomService(localMockRoomRepo)
	localUserService := user.NewUserService(mockUserRepo)
	gameService := NewGameService(localMockGameRepo, localMockRoomRepo, localRoomService, localUserService)

	localMockRoomRepo.On("GetRoom", "roomX").Return(nil, assert.AnError)
	err := gameService.StartGame("host", "roomX", true)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestStartGameFailsOnValidateRoom(t *testing.T) {
	localMockGameRepo := new(MockGameStateRepository)
	localMockRoomRepo := new(MockRoomRepository)
	localRoomService := NewRoomService(localMockRoomRepo)
	localUserService := user.NewUserService(mockUserRepo)
	gameService := NewGameService(localMockGameRepo, localMockRoomRepo, localRoomService, localUserService)

	room := &Room{ID: "roomY", Host: Player{ID: "host"}, Status: "LOBBY", Team1: []Player{}, Team2: []Player{}}
	localMockRoomRepo.On("GetRoom", "roomY").Return(room, nil)

	err := gameService.StartGame("host", "roomY", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enough players")
}
