package game

import (
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
	"github.com/thesrcielos/TopTankBattle/internal/user"
)

type GameStateRepositoryMock struct {
	mock.Mock
}

func (m *GameStateRepositoryMock) PublishToRoom(payload string) {
	m.Called(payload)
}

func (m *GameStateRepositoryMock) SubscribeMessages() error {
	args := m.Called()
	return args.Error(0)
}

func (m *GameStateRepositoryMock) SendReceivedMessage(messageEncoded string) {
	m.Called(messageEncoded)
}

func (m *GameStateRepositoryMock) TryToBecomeLeader(roomID string) bool {
	args := m.Called(roomID)
	return args.Bool(0)
}

func (m *GameStateRepositoryMock) SaveGameState(gameState *state.GameState) {
	m.Called(gameState)
}

func (m *GameStateRepositoryMock) RestoreGameState(roomID string) *state.GameState {
	args := m.Called(roomID)
	return args.Get(0).(*state.GameState)
}

func (m *GameStateRepositoryMock) RenewLeadership(roomID string, expiration time.Duration) (bool, error) {
	args := m.Called(roomID, expiration)
	return args.Bool(0), args.Error(1)
}

func (m *GameStateRepositoryMock) UpdateGamePlayerState(playerId string, position state.Position) {
	m.Called(playerId, position)
}

func (m *GameStateRepositoryMock) UpdateGameBullets(bullet state.Bullet) {
	m.Called(bullet)
}

type RoomRepositoryMock struct {
	mock.Mock
}

func (m *RoomRepositoryMock) SaveRoomRequest(req *RoomRequest) (*Room, error) {
	args := m.Called(req)
	return args.Get(0).(*Room), args.Error(1)
}

func (m *RoomRepositoryMock) SaveRoom(room *Room) error {
	args := m.Called(room)
	return args.Error(0)
}

func (m *RoomRepositoryMock) SavePlayerRoom(req *PlayerRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *RoomRepositoryMock) GetPlayerRoom(playerId string) (interface{}, error) {
	args := m.Called(playerId)
	return args.Get(0), args.Error(1)
}

func (m *RoomRepositoryMock) DeletePlayerRoom(playerId string) error {
	args := m.Called(playerId)
	return args.Error(0)
}

func (m *RoomRepositoryMock) GetRoom(key string) (*Room, error) {
	args := m.Called(key)
	return args.Get(0).(*Room), args.Error(1)
}

func (m *RoomRepositoryMock) GetRooms(page, pageSize int) (*[]Room, error) {
	args := m.Called(page, pageSize)
	return args.Get(0).(*[]Room), args.Error(1)
}

func (m *RoomRepositoryMock) CreatePlayer(id int) (*Player, error) {
	args := m.Called(id)
	return args.Get(0).(*Player), args.Error(1)
}

func (m *RoomRepositoryMock) AddPlayer(req *PlayerRequest) (*Room, error) {
	args := m.Called(req)
	return args.Get(0).(*Room), args.Error(1)
}

func (m *RoomRepositoryMock) RemovePlayer(req *PlayerRequest) (*Room, error) {
	args := m.Called(req)
	return args.Get(0).(*Room), args.Error(1)
}

func (m *RoomRepositoryMock) ChangeRoomOwner(roomId string, player Player) (*Room, error) {
	args := m.Called(roomId, player)
	return args.Get(0).(*Room), args.Error(1)
}

func (m *RoomRepositoryMock) DeleteRoom(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *RoomRepositoryMock) PublishToRoom(payload string) {
	m.Called(payload)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(username, password string) (*user.User, error) {
	args := m.Called(username, password)
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) ValidateUser(username, password string) (*user.User, error) {
	args := m.Called(username, password)
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetUserUsername(id int) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

func (m *MockUserRepository) GetUser(id int) (*user.User, error) {
	args := m.Called(id)
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) FetchUserStats(userID int) (user.UserStats, error) {
	args := m.Called(userID)
	return args.Get(0).(user.UserStats), args.Error(1)
}

func (m *MockUserRepository) UpdateUserStats(stats *user.UserStats) error {
	args := m.Called(stats)
	return args.Error(0)
}
