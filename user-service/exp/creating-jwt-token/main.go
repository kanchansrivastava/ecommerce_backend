package main

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"os"
	"time"
)

func main() {

	privateKeyPem, err := os.ReadFile("private.pem")
	if err != nil {
		panic(err)
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPem)

	// iss (issuer): Issuer of the JWT
	// sub (subject): Subject of the JWT (the users)
	// aud (audience): Recipient for which the JWT is intended
	// exp (expiration time): Time after which the JWT expires
	// nbf (not before time): Time before which the JWT must not be accepted for processing
	// iat (issued at time): Time at which the JWT was issued; can be used to determine age of the JWT
	// jti (JWT ID): Unique identifier; can be used to prevent the JWT from being replayed (allows a token to be used only once)
	claims := struct {
		jwt.RegisteredClaims
		Roles []string `json:"roles"`
	}{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "user-service",
			Subject:   "101",                                                // userId
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(50 * time.Minute)), // after 50 minutes, this token expires
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Roles: []string{"USER"},
	}

	//SigningMethodRSA implements the RSA family of signing methods.
	//Expects *rsa.PrivateKey for signing and *rsa.PublicKey for validation
	tkn := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	str, err := tkn.SignedString(privateKey)
	fmt.Println(str)
}
