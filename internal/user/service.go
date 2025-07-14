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
