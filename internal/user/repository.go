package user

import (
	"errors"

	"github.com/thesrcielos/TopTankBattle/internal/apperrors"
	"github.com/thesrcielos/TopTankBattle/pkg/db"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func CreateUser(username, password string) (*User, error) {
	var newUser *User

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var exists User
		result := tx.Where("username = ?", username).First(&exists)
		if result.Error == nil {
			return apperrors.NewAppError(409, "User already exists", errors.New("username already exists"))
		}
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return apperrors.NewAppError(500, "Error retrieving user", result.Error)
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(password), 14)
		if err != nil {
			return apperrors.NewAppError(500, "Error hashing password", err)
		}

		newUser = &User{
			Username: username,
			Password: string(hashed),
		}
		if err := tx.Create(newUser).Error; err != nil {
			return apperrors.NewAppError(500, "Error creating user", err)
		}

		stats := UserStats{
			UserID:      newUser.ID,
			TotalGames:  0,
			TotalWins:   0,
			TotalLosses: 0,
		}
		if err := tx.Create(&stats).Error; err != nil {
			return apperrors.NewAppError(500, "Error creating user stats", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return newUser, nil
}

func ValidateUser(username, password string) (*User, error) {
	var u User
	result := db.DB.Where("username = ?", username).First(&u)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewAppError(404, "User not found", errors.New("no stats for user"))
		} else {
			return nil, apperrors.NewAppError(500, "Error retrieving user", result.Error)
		}
	}
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		return nil, apperrors.NewAppError(400, "Invalid password", err)
	}

	return &u, nil
}

func GetUserUsername(id int) (string, error) {
	var u User
	result := db.DB.Where("id = ?", id).First(&u)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", apperrors.NewAppError(404, "User not found", errors.New("no stats for user"))
		} else {
			return "", apperrors.NewAppError(500, "Error retrieving user", result.Error)
		}
	}
	return u.Username, nil
}

func GetUser(id int) (*User, error) {
	var u User
	result := db.DB.Where("id = ?", id).First(&u)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewAppError(404, "User not found", errors.New("no stats for user"))
		} else {
			return nil, apperrors.NewAppError(500, "Error retrieving user", result.Error)
		}
	}

	return &u, nil
}

func FetchUserStats(userID int) (UserStats, error) {
	var stats UserStats
	result := db.DB.Where("user_id = ?", userID).First(&stats)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return UserStats{}, apperrors.NewAppError(404, "User stats not found", errors.New("no stats for user"))
		} else {
			return UserStats{}, apperrors.NewAppError(500, "Error retrieving user stats", result.Error)
		}
	}
	return stats, nil
}
