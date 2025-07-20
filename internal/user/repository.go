package user

import (
	"errors"

	"github.com/thesrcielos/TopTankBattle/internal/apperrors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(username, password string) (*User, error)
	ValidateUser(username, password string) (*User, error)
	GetUserUsername(id int) (string, error)
	GetUser(id int) (*User, error)
	FetchUserStats(userID int) (UserStats, error)
	UpdateUserStats(stats *UserStats) error
}

type UserRepositoryImpl struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &UserRepositoryImpl{
		db: db,
	}
}

const ERROR_RETRIEVING_USER = "Error retrieving user"
const ERROR_USER_NOT_FOUND = "User not found"

func (u *UserRepositoryImpl) CreateUser(username, password string) (*User, error) {
	var newUser *User

	err := u.db.Transaction(func(tx *gorm.DB) error {
		var exists User
		result := tx.Where("username = ?", username).First(&exists)
		if result.Error == nil {
			return apperrors.NewAppError(409, "User already exists", errors.New("username already exists"))
		}
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return apperrors.NewAppError(500, ERROR_RETRIEVING_USER, result.Error)
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

func (u *UserRepositoryImpl) ValidateUser(username, password string) (*User, error) {
	var user User
	result := u.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewAppError(404, ERROR_USER_NOT_FOUND, nil)
		} else {
			return nil, apperrors.NewAppError(500, ERROR_RETRIEVING_USER, result.Error)
		}
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, apperrors.NewAppError(400, "Invalid password", err)
	}

	return &user, nil
}

func (u *UserRepositoryImpl) GetUserUsername(id int) (string, error) {
	var user User
	result := u.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", apperrors.NewAppError(404, ERROR_USER_NOT_FOUND, nil)
		} else {
			return "", apperrors.NewAppError(500, ERROR_RETRIEVING_USER, result.Error)
		}
	}
	return user.Username, nil
}

func (u *UserRepositoryImpl) GetUser(id int) (*User, error) {
	var user User
	result := u.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewAppError(404, ERROR_USER_NOT_FOUND, nil)
		} else {
			return nil, apperrors.NewAppError(500, ERROR_RETRIEVING_USER, result.Error)
		}
	}

	return &user, nil
}

func (u *UserRepositoryImpl) FetchUserStats(userID int) (UserStats, error) {
	var stats UserStats
	result := u.db.Where("user_id = ?", userID).First(&stats)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return UserStats{}, apperrors.NewAppError(404, "User stats not found", nil)
		} else {
			return UserStats{}, apperrors.NewAppError(500, ERROR_RETRIEVING_USER, result.Error)
		}
	}
	return stats, nil
}

func (u *UserRepositoryImpl) UpdateUserStats(stats *UserStats) error {
	result := u.db.Save(stats)
	if result.Error != nil {
		return apperrors.NewAppError(500, ERROR_RETRIEVING_USER, result.Error)
	}
	return nil
}
