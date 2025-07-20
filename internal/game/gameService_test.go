package game

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thesrcielos/TopTankBattle/internal/game/maps"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
	"github.com/thesrcielos/TopTankBattle/internal/user"
)

var (
	mockRoomRepo RoomRepository
	mockGameRepo GameStateRepository
	mockUserRepo user.UserRepository

	roomService *RoomService
	userService *user.UserService
)

func TestMain(m *testing.M) {
	// Mock Repositories
	mockRoomRepo = new(RoomRepositoryMock)
	mockGameRepo = new(GameStateRepositoryMock)
	mockUserRepo = new(MockUserRepository)

	roomService = NewRoomService(mockRoomRepo)
	userService = user.NewUserService(mockUserRepo)

	code := m.Run()
	os.Exit(code)
}

func TestValidateRoom_Ok(t *testing.T) {
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
