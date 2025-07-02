package user

import (
	"errors"

	"github.com/thesrcielos/TopTankBattle/pkg/db"
	"golang.org/x/crypto/bcrypt"
)

func CreateUser(username, password string) (*User, error) {
	var exists User
	result := db.DB.Where("username = ?", username).First(&exists)
	if result.Error == nil {
		return nil, errors.New("user already exists")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return nil, err
	}
	newUser := User{
		Username: username,
		Password: string(hashed),
	}

	if err := db.DB.Create(&newUser).Error; err != nil {
		return nil, err
	}

	return &newUser, nil
}

func ValidateUser(username, password string) (*User, error) {
	var u User
	result := db.DB.Where("username = ?", username).First(&u)
	if result.Error != nil {
		return nil, result.Error
	}
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func GetUserUsername(id string) (string, error) {
	var u User
	result := db.DB.Where("id = ?", id).First(&u)
	if result.Error != nil {
		return "", result.Error
	}

	return u.Username, nil
}
