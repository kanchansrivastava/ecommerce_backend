package middleware

import (
	"errors"
	"user-service/internal/auth"
)

type Mid struct {
	a *auth.Keys
}

func NewMid(a *auth.Keys) (Mid, error) {
	if a == nil {
		return Mid{}, errors.New("keys must not be nil")
	}
	return Mid{a: a}, nil
}
