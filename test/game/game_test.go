package game_test

import (
	"testing"

	"github.com/thesrcielos/TopTankBattle/internal/game"
	"github.com/thesrcielos/TopTankBattle/internal/game/maps"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
)

var dummyObstacles = [][]bool{
	{false, false, false, false},
	{false, false, false, false},
	{false, false, false, false},
	{false, false, false, false},
}

func TestCheckBulletCollisionHitsPlayer(t *testing.T) {
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
	pHit, fHit, destroyed := game.CheckBulletCollision(bullet, players, fortresses)

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

	p, f, destroyed := game.CheckBulletCollision(bullet, players, nil)
	if !destroyed {
		t.Errorf("Expected bullet to be destroyed by obstacle")
	}
	if p != nil || f != nil {
		t.Errorf("Expected no player or fortress hit")
	}
}

func TestCheckBulletCollisionHitsAllyPlayer(t *testing.T) {
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

	p, f, destroyed := game.CheckBulletCollision(bullet, players, nil)
	if destroyed == false {
		t.Errorf("Expected bullet to be destroyed by ally collision")
	}
	if p != nil || f != nil {
		t.Errorf("Expected no kill, only destruction")
	}
}

func TestCheckBulletCollisionHitsEnemyFortress(t *testing.T) {
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

	p, f, destroyed := game.CheckBulletCollision(bullet, players, fortresses)
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
