package user

import (
	"errors"

	"github.com/thesrcielos/TopTankBattle/internal/apperrors"
)

func Signup(user User) (string, error) {
	u, err := CreateUser(user.Username, user.Password)
	if err != nil {
		return "", err
	}

	token, errJWT := GenerateJWT(u.ID)
	if errJWT != nil {
		return "", apperrors.NewAppError(500, "error creating jwt token", errJWT)
	}
	return token, nil
}

func Login(user User) (string, error) {
	u, err := ValidateUser(user.Username, user.Password)
	if err != nil {
		return "", errors.New("invalid credentials")
	}
	token, errJWT := GenerateJWT(u.ID)
	if errJWT != nil {
		return "", apperrors.NewAppError(500, "error creating jwt token", errJWT)
	}
	return token, nil
}

func GetUserStats(userID int) (*UserStatsResponse, error) {
	user, erruserID := GetUser(userID)
	if erruserID != nil {
		return nil, erruserID
	}

	if user == nil {
		return nil, apperrors.NewAppError(404, "user not found", errors.New("user not found"))
	}

	stats, err := FetchUserStats(userID)
	if err != nil {
		return nil, err
	}

	winRate := 0.0
	if stats.TotalGames > 0 {
		winRate = 100 * (float64(stats.TotalWins) / float64(stats.TotalGames))
	}

	response := &UserStatsResponse{
		Username:   user.Username,
		TotalGames: stats.TotalGames,
		Wins:       stats.TotalWins,
		Losses:     stats.TotalLosses,
		WinRate:    winRate,
	}
	return response, nil
}
