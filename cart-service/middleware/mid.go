package middleware

import "cart-service/internal/auth"

type Mid struct {
	k *auth.Keys
}

func NewMid(k *auth.Keys) *Mid {
	return &Mid{k: k}
}
