package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestRoomService(t *testing.T) (*RoomService, *MockRoomRepository) {
	mockRepo := NewMockRoomRepository(t)
	rs := NewRoomService(mockRepo)
	return rs, mockRepo
}

func TestRoomServiceCreateRoomSuccess(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	roomReq := &RoomRequest{Name: "TestRoom", Player: 1, Capacity: 2}
	mockRepo.On("GetPlayerRoom", "1").Return(nil, nil)
	room := &Room{ID: "room1", Host: Player{ID: "1"}}
	mockRepo.On("SaveRoomRequest", roomReq).Return(room, nil)
	mockRepo.On("SavePlayerRoom", mock.AnythingOfType("*game.PlayerRequest")).Return(nil)

	result, err := rs.CreateRoom(roomReq)
	assert.NoError(t, err)
	assert.Equal(t, room, result)
	mockRepo.AssertExpectations(t)
}

func TestRoomServiceCreateRoomPlayerAlreadyInRoom(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	roomReq := &RoomRequest{Name: "TestRoom", Player: 1, Capacity: 2}
	mockRepo.On("GetPlayerRoom", "1").Return("room1", nil)

	result, err := rs.CreateRoom(roomReq)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Player already in a room")
	mockRepo.AssertExpectations(t)
}

func TestRoomServiceJoinRoomSuccess(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	playerReq := &PlayerRequest{Player: "2", Room: "room1"}
	mockRepo.On("GetPlayerRoom", "2").Return(nil, nil)
	roomJoin := &Room{ID: "room1", Host: Player{ID: "1"}, Team1: []Player{{ID: "1"}}, Team2: []Player{{ID: "2"}}, Players: 2}
	mockRepo.On("AddPlayer", playerReq).Return(roomJoin, nil)
	mockRepo.On("SavePlayerRoom", playerReq).Return(nil)
	mockRepo.On("PublishToRoom", mock.Anything).Return()

	result, err := rs.JoinRoom(playerReq)
	assert.NoError(t, err)
	assert.Equal(t, roomJoin, result)
	mockRepo.AssertExpectations(t)
}

func TestRoomServiceJoinRoomAlreadyInRoom(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	playerReq := &PlayerRequest{Player: "2", Room: "room1"}
	mockRepo.On("GetPlayerRoom", "2").Return("room1", nil)

	result, err := rs.JoinRoom(playerReq)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Player already in a room")
	mockRepo.AssertExpectations(t)
}

func TestRoomServiceLeaveRoomSuccess(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	mockRepo.On("GetPlayerRoom", "2").Return("room1", nil)
	playerReq := &PlayerRequest{Player: "2", Room: "room1"}
	room := &Room{ID: "room1", Host: Player{ID: "1"}, Team1: []Player{{ID: "1"}}, Team2: []Player{}, Players: 1}
	mockRepo.On("RemovePlayer", playerReq).Return(room, nil)
	mockRepo.On("DeletePlayerRoom", "2").Return(nil)
	mockRepo.On("PublishToRoom", mock.Anything).Return()

	err := rs.LeaveRoom("2")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRoomServiceLeaveRoomNotInRoom(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	mockRepo.On("GetPlayerRoom", "2").Return(nil, nil)

	err := rs.LeaveRoom("2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Player is not in a room")
	mockRepo.AssertExpectations(t)
}

func TestRoomServiceChangeOwnerIfNeededChangesOwner(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	room := &Room{
		ID:      "room1",
		Host:    Player{ID: "oldhost"},
		Players: 2,
		Team1:   []Player{{ID: "newhost"}},
		Team2:   []Player{},
	}
	mockRepo.On("ChangeRoomOwner", "room1", room.Team1[0]).Return(room, nil)
	err := rs.changeOwnerIfNeeded("oldhost", room)
	assert.NoError(t, err)
	assert.Equal(t, "newhost", room.Host.ID)
	mockRepo.AssertExpectations(t)
}

func TestRoomServiceChangeOwnerIfNeededNoChangeIfNotHostOrNoPlayers(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	room := &Room{
		ID:      "room1",
		Host:    Player{ID: "host"},
		Players: 0,
		Team1:   []Player{},
		Team2:   []Player{},
	}
	// No se debe llamar a ChangeRoomOwner
	err := rs.changeOwnerIfNeeded("notTheHost", room)
	assert.NoError(t, err)
	err = rs.changeOwnerIfNeeded("host", room)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRoomServiceGetRoomsSuccess(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	rooms := []Room{{ID: "room1"}, {ID: "room2"}}
	mockRepo.On("GetRooms", 1, 10).Return(&rooms, nil)
	result, err := rs.GetRooms(&RoomPageRequest{Page: 1, PageSize: 10})
	assert.NoError(t, err)
	assert.Equal(t, &rooms, result)
	mockRepo.AssertExpectations(t)
}

func TestRoomServiceGetRoomsError(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	mockRepo.On("GetRooms", 1, 10).Return(nil, assert.AnError)
	result, err := rs.GetRooms(&RoomPageRequest{Page: 1, PageSize: 10})
	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestRoomRequestValidate(t *testing.T) {
	r := &RoomRequest{Name: "Sala", Player: 1, Capacity: 3}
	err := r.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "capacity must be 2, 4, or 6")

	r = &RoomRequest{Name: "abcdefghijklmnopqrstuvwxyz1234567890", Player: 1, Capacity: 2}
	err = r.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name must not exceed 30 characters")

	r = &RoomRequest{Name: "SalaValida", Player: 1, Capacity: 4}
	err = r.Validate()
	assert.NoError(t, err)
}

func TestRoomServiceGetPlayerFromRoom(t *testing.T) {
	rs, _ := newTestRoomService(t)
	room := &Room{
		ID:    "room1",
		Team1: []Player{{ID: "p1"}},
		Team2: []Player{{ID: "p2"}},
	}

	result, err := rs.getPlayerFromRoom("p1", room)
	assert.NoError(t, err)
	assert.Equal(t, Player{ID: "p1"}, result["player"])
	assert.Equal(t, 1, result["team"])

	result, err = rs.getPlayerFromRoom("p2", room)
	assert.NoError(t, err)
	assert.Equal(t, Player{ID: "p2"}, result["player"])
	assert.Equal(t, 2, result["team"])

	result, err = rs.getPlayerFromRoom("p3", room)
	assert.Error(t, err)
	assert.Nil(t, result)
}
