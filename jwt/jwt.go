package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-jwt/jwt/v5/request"
)

type Claims struct {
	UserID int `json:"userId"`
	jwt.RegisteredClaims
}

var key *ecdsa.PrivateKey = newKey()

func newKey() *ecdsa.PrivateKey {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	return key
}

func IssueToken(userID int) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(key)
}

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := request.ParseFromRequest(
			c.Request,
			request.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}

				return key.Public(), nil
			},
			request.WithClaims(&Claims{}),
		)
		if err != nil || token == nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
			})
			c.Abort()
			return
		}

		c.Set("userID", token.Claims.(*Claims).UserID)
		c.Next()
	}
}

func GetCurrentUserID(c *gin.Context) (int, bool) {
	userID, ok := c.Get("userID")
	return userID.(int), ok
}
