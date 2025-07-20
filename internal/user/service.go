package user

import (
	"errors"
	"fmt"

	"github.com/thesrcielos/TopTankBattle/internal/apperrors"
)

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (u *UserService) Signup(user User) (string, error) {
	userRetrieved, err := u.repo.CreateUser(user.Username, user.Password)
	if err != nil {
		return "", err
	}

	token, errJWT := GenerateJWT(userRetrieved.ID)
	if errJWT != nil {
		return "", apperrors.NewAppError(500, "error creating jwt token", errJWT)
	}
	return token, nil
}

func (u *UserService) Login(user User) (string, error) {
	userRetrieved, err := u.repo.ValidateUser(user.Username, user.Password)
	if err != nil {
		return "", errors.New("invalid credentials")
	}
	token, errJWT := GenerateJWT(userRetrieved.ID)
	if errJWT != nil {
		return "", apperrors.NewAppError(500, "error creating jwt token", errJWT)
	}
	return token, nil
}

func (u *UserService) GetUserStats(userID int) (*UserStatsResponse, error) {
	user, erruserID := u.repo.GetUser(userID)
	if erruserID != nil {
		return nil, erruserID
	}

	if user == nil {
		return nil, apperrors.NewAppError(404, "user not found", errors.New("user not found"))
	}

	stats, err := u.repo.FetchUserStats(userID)
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
	fmt.Println("User stats:", response)
	return response, nil
}

func (u *UserService) UpdatePlayerStats(userID int, win bool) error {
	stats, err := u.repo.FetchUserStats(userID)
	if err != nil {
		return err
	}

	if win {
		stats.TotalWins++
	} else {
		stats.TotalLosses++
	}
	stats.TotalGames++

	if err := u.repo.UpdateUserStats(&stats); err != nil {
		return apperrors.NewAppError(500, "error updating user stats", err)
	}

	return nil
}
