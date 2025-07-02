package user

import (
	"errors"
)

func Signup(user User) (string, error) {
	u, err := CreateUser(user.Username, user.Password)
	if err != nil {
		return "", err
	}

	token, errJWT := GenerateJWT(u.ID)
	if errJWT != nil {
		return "", errJWT
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
		return "", errJWT
	}
	return token, nil
}
