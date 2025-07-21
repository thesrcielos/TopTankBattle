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

func TestRoomService_CreateRoom_Success(t *testing.T) {
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

func TestRoomService_CreateRoom_PlayerAlreadyInRoom(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	roomReq := &RoomRequest{Name: "TestRoom", Player: 1, Capacity: 2}
	mockRepo.On("GetPlayerRoom", "1").Return("room1", nil)

	result, err := rs.CreateRoom(roomReq)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Player already in a room")
	mockRepo.AssertExpectations(t)
}

func TestRoomService_JoinRoom_Success(t *testing.T) {
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

func TestRoomService_JoinRoom_AlreadyInRoom(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	playerReq := &PlayerRequest{Player: "2", Room: "room1"}
	mockRepo.On("GetPlayerRoom", "2").Return("room1", nil)

	result, err := rs.JoinRoom(playerReq)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Player already in a room")
	mockRepo.AssertExpectations(t)
}

func TestRoomService_LeaveRoom_Success(t *testing.T) {
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

func TestRoomService_LeaveRoom_NotInRoom(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	mockRepo.On("GetPlayerRoom", "2").Return(nil, nil)

	err := rs.LeaveRoom("2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Player is not in a room")
	mockRepo.AssertExpectations(t)
}

func TestRoomService_KickPlayerFromRoom_Success(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	room := &Room{ID: "room1", Host: Player{ID: "1"}, Team1: []Player{{ID: "1"}, {ID: "2"}}, Team2: []Player{}, Players: 2}
	mockRepo.On("GetRoom", "room1").Return(room, nil)
	mockRepo.On("GetPlayerRoom", "2").Return("room1", nil)
	playerReq := &PlayerRequest{Player: "2", Room: "room1"}
	mockRepo.On("RemovePlayer", playerReq).Return(room, nil)
	mockRepo.On("DeletePlayerRoom", "2").Return(nil)
	mockRepo.On("PublishToRoom", mock.Anything).Return()

	result, err := rs.KickPlayerFromRoom("1", "room1", "2")
	assert.NoError(t, err)
	assert.Equal(t, room, result)
	mockRepo.AssertExpectations(t)
}

func TestRoomService_KickPlayerFromRoom_NotHost(t *testing.T) {
	rs, mockRepo := newTestRoomService(t)
	room := &Room{ID: "room1", Host: Player{ID: "1"}, Team1: []Player{{ID: "1"}, {ID: "2"}}, Team2: []Player{}, Players: 2}
	mockRepo.On("GetRoom", "room1").Return(room, nil)
	mockRepo.On("GetPlayerRoom", "2").Return("room1", nil)

	result, err := rs.KickPlayerFromRoom("2", "room1", "2")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Only the host can kick players")
	mockRepo.AssertExpectations(t)
}
