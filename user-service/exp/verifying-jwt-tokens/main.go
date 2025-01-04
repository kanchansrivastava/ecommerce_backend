package main

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"os"
)

var tokenStr = `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ1c2VyLXNlcnZpY2UiLCJzdWIiOiIxMDEiLCJleHAiOjE3MzUyMDc4MzYsImlhdCI6MTczNTIwNDgzNiwicm9sZXMiOlsiVVNFUiJdfQ.tvMznL6iEaNDa-0d1_V1gjoDa7mta2yU6x4ovc6OR82d_OQdGiD5CHXxYC7tCGR12OtidsNmuoz9s030TnAMwGN-ImfHzv-SOonv4Hozq31ZguPRmf-ciNnHhIU-ByrJLtui12HOlNJwV7WQQjqOTPch-DOPSAz5hlHTb1R2M8EFlZU5E-rD61Mwpz0w2H9kqkcgxxZMMIxgVcxS_2OUkn1MX5Dwet1IraoUxaU5YjmLc7Lnal43Wi0ejnQP-0_Uro6TSzLuM8Uxg8n_hlRy8xpf3ijMQY0SE0-wJzuPr9rSkbZUuc10o1bPP2p6ftyk0eWPn9rAP8TKJ2FbkxQfBQ`

func main() {
	pubKeyPem, err := os.ReadFile("pubkey.pem")
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubKeyPem)

	if err != nil {
		panic(err)
	}
	var claims struct {
		jwt.RegisteredClaims
		Roles []string `json:"roles"`
	}
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (interface{}, error) {
		return pubKey, nil
	})

	if err != nil {
		panic(err)
	}
	if !token.Valid {
		panic("token is not valid")
	}
	fmt.Println(claims.Subject, ", this user requested this role:", claims.Roles)
}
