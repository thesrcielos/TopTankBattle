package user

import (
	"errors"

	"github.com/thesrcielos/TopTankBattle/internal/apperrors"
	"github.com/thesrcielos/TopTankBattle/pkg/db"
	"golang.org/x/crypto/bcrypt"
)

func CreateUser(username, password string) (*User, error) {
	var exists User
	result := db.DB.Where("username = ?", username).First(&exists)
	if result.Error == nil {
		return nil, apperrors.NewAppError(404, "User already exists", errors.New("username already exists"))
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return nil, apperrors.NewAppError(500, "Error hashing password", err)
	}
	newUser := User{
		Username: username,
		Password: string(hashed),
	}

	if err := db.DB.Create(&newUser).Error; err != nil {
		return nil, apperrors.NewAppError(500, "Error creating user", err)
	}

	return &newUser, nil
}

func ValidateUser(username, password string) (*User, error) {
	var u User
	result := db.DB.Where("username = ?", username).First(&u)
	if result.Error != nil {
		return nil, apperrors.NewAppError(500, "Error retrieving user", result.Error)
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
		return "", apperrors.NewAppError(500, "Error retrieving user", result.Error)
	}

	return u.Username, nil
}
