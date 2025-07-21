package user

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockGenerateJWT is a helper to override GenerateJWT in tests
var mockGenerateJWT func(id uint) (string, error)

func TestMain(m *testing.M) {
	// Patch GenerateJWT for all tests
	orig := GenerateJWT
	GenerateJWT = func(id uint) (string, error) {
		if mockGenerateJWT != nil {
			return mockGenerateJWT(id)
		}
		return orig(id)
	}
	code := m.Run()
	GenerateJWT = orig
	os.Exit(code)
}

func TestUserService_Signup(t *testing.T) {
	mockRepo := &MockUserRepository{}
	service := NewUserService(mockRepo)

	user := User{ID: 1, Username: "test", Password: "pass"}
	mockRepo.On("CreateUser", user.Username, user.Password).Return(&user, nil)
	mockGenerateJWT = func(id uint) (string, error) { return "token123", nil }

	token, err := service.Signup(user)
	assert.NoError(t, err)
	assert.Equal(t, "token123", token)
	mockRepo.AssertExpectations(t)
}

func TestUserService_Login(t *testing.T) {
	mockRepo := &MockUserRepository{}
	service := NewUserService(mockRepo)

	user := User{ID: 2, Username: "foo", Password: "bar"}
	mockRepo.On("ValidateUser", user.Username, user.Password).Return(&user, nil)
	mockGenerateJWT = func(id uint) (string, error) { return "tok456", nil }

	token, err := service.Login(user)
	assert.NoError(t, err)
	assert.Equal(t, "tok456", token)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetUserStats(t *testing.T) {
	mockRepo := &MockUserRepository{}
	service := NewUserService(mockRepo)

	user := &User{ID: 3, Username: "alice"}
	stats := UserStats{UserID: 3, TotalGames: 10, TotalWins: 7, TotalLosses: 3}
	mockRepo.On("GetUser", 3).Return(user, nil)
	mockRepo.On("FetchUserStats", 3).Return(stats, nil)

	resp, err := service.GetUserStats(3)
	assert.NoError(t, err)
	assert.Equal(t, "alice", resp.Username)
	assert.Equal(t, 10, resp.TotalGames)
	assert.Equal(t, 7, resp.Wins)
	assert.Equal(t, 3, resp.Losses)
	assert.InDelta(t, 70.0, resp.WinRate, 0.01)
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdatePlayerStats(t *testing.T) {
	mockRepo := &MockUserRepository{}
	service := NewUserService(mockRepo)

	stats := UserStats{UserID: 4, TotalGames: 2, TotalWins: 1, TotalLosses: 1}
	mockRepo.On("FetchUserStats", 4).Return(stats, nil)
	mockRepo.On("UpdateUserStats", mock.AnythingOfType("*user.UserStats")).Return(nil)

	err := service.UpdatePlayerStats(4, true)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_Signup_Error(t *testing.T) {
	mockRepo := &MockUserRepository{}
	service := NewUserService(mockRepo)
	user := User{ID: 5, Username: "err", Password: "fail"}
	mockRepo.On("CreateUser", user.Username, user.Password).Return(nil, errors.New("fail"))

	_, err := service.Signup(user)
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}
