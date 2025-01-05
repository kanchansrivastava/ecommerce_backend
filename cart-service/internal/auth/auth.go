package auth

import (
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey int

const ClaimsKey ctxKey = 1

const RoleUser = "user"
const RoleAdmin = "admin"

type Keys struct {
	publicKey *rsa.PublicKey
}

// NewKeys is a constructor function for Keys struct. It accepts privateKey and publicKey as parameters and returns
// an instance of Keys struct. If either of privateKey or publicKey is nil, it returns an error.
func NewKeys(publicKey *rsa.PublicKey) (*Keys, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("invalid keys")
	}
	return &Keys{publicKey}, nil

}

type Claims struct {
	jwt.RegisteredClaims
	Roles []string `json:"roles"`
}

func (c Claims) HasRoles(requiredRoles ...string) bool {
	for _, has := range c.Roles { // roles with the user in the token
		for _, want := range requiredRoles {
			if has == want {
				return true
			}
		}
	}
	return false
}

// ValidateToken is a method for Auth struct. It verifies the provided JWT token using the publicKey of the Auth struct
// it's called upon and returns the parsed claims if the JWT token is valid. If the JWT token is invalid or
// there is an error during parsing, it returns an error.
func (k *Keys) ValidateToken(tokenStr string) (Claims, error) {
	var claims Claims
	tkn, err := jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (interface{}, error) {
		return k.publicKey, nil
	})
	if err != nil {
		return Claims{}, err
	}
	if !tkn.Valid {
		return Claims{}, fmt.Errorf("invalid token")
	}
	return claims, nil
}
