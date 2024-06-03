package controller

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"disgord/ent/user"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-jwt/jwt/v5/request"
	"golang.org/x/crypto/bcrypt"
)

// SignUp godoc
//
//	@Tags		auth
//	@Summary	sign up and create a new user
//	@Param		body	body		controller.SignUp.Body	true	"Request body"
//	@Success	201		{object}	ent.User
//	@Failure	409		"username already exists"
//	@Router		/auth/sign-up [post]
func (*Controller) SignUp(c *gin.Context) {
	type Body struct {
		Username    string `json:"username" binding:"required"`
		Password    string `json:"password" binding:"required"`
		DisplayName string `json:"displayName" binding:"required"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		return
	}

	user, err := client.User.
		Create().
		SetUsername(body.Username).
		SetPassword(hashPassword(body.Password)).
		SetDisplayName(body.DisplayName).
		Save(ctx)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"message": "username already exists",
		})
		return
	}

	c.JSON(http.StatusCreated, user)
}

type Token struct {
	AccessToken string `json:"accessToken"`
}

// SignIn godoc
//
//	@Description	Set "Authorization" header with the "Bearer ${accessToken}" to authenticate requests.
//	@Tags			auth
//	@Summary		sign in and receive an access token
//	@Param			body	body		controller.SignIn.Body	true	"Request body"
//	@Success		200		{object}	controller.Token
//	@Failure		401		"invalid username or password"
//	@Failure		404		"user not found"
//	@Router			/auth/sign-in [post]
func (*Controller) SignIn(c *gin.Context) {
	type Body struct {
		Username string `binding:"required"`
		Password string `binding:"required"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		return
	}

	user, err := client.User.
		Query().
		Where(user.Username(body.Username)).
		Only(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "user not found",
		})
		return
	}

	if !verifyPassword(user.Password, body.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "invalid username or password",
		})
		return
	}

	tokenString, err := issueToken(user.ID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.JSON(http.StatusOK, Token{
		AccessToken: tokenString,
	})
}

var key *ecdsa.PrivateKey

func init() {
	b, err := os.ReadFile("disgord.pem")
	if err != nil {
		log.Fatal(err)
	}

	key, err = jwt.ParseECPrivateKeyFromPEM(b)
	if err != nil {
		log.Fatal(err)
	}
}

type Claims struct {
	UserID int `json:"userId"`
	jwt.RegisteredClaims
}

func (*Controller) JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := request.ParseFromRequest(
			c.Request,
			request.OAuth2Extractor,
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
				"message": "unauthorized",
			})
			c.Abort()
			return
		}

		userID := token.Claims.(*Claims).UserID

		c.Set("userID", userID)
		c.Next()
	}
}

func getCurrentUserID(c *gin.Context) int {
	userID, _ := c.Get("userID")
	return userID.(int)
}

func issueToken(userID int) (string, error) {
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

func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	return string(hash)
}

func verifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
