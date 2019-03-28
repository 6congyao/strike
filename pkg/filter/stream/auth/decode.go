package auth

import (
	"github.com/dgrijalva/jwt-go"
)

// return token payload
func decode(pubKey, tokenString string) (map[string]interface{}, error) {
	key, e := jwt.ParseRSAPublicKeyFromPEM([]byte(pubKey))
	if e != nil {
		panic(e.Error())
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	payload := token.Claims.(jwt.MapClaims)
	return payload, payload.Valid()
}
