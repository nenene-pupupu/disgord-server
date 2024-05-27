package controller

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"time"

	"disgord/ent/auth"
	"disgord/ent/user"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-jwt/jwt/v5/request"
	"golang.org/x/crypto/bcrypt"
)

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

	auth, err := client.Auth.
		Query().
		Where(auth.HasUserWith(
			user.Username(body.Username),
		)).
		Only(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "user not found",
		})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(auth.Password), []byte(body.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "invalid username or password",
		})
		return
	}

	tokenString, err := issueToken(auth.UserID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.JSON(http.StatusOK, Token{
		AccessToken: tokenString,
	})
}

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

	_, err := client.User.
		Query().
		Where(user.Username(body.Username)).
		Only(ctx)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"message": "username already exists",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	tx, err := client.Tx(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer tx.Rollback()

	user, err := tx.User.
		Create().
		SetUsername(body.Username).
		SetDisplayName(body.DisplayName).
		Save(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	_, err = tx.Auth.
		Create().
		SetUser(user).
		SetPassword(string(hash)).
		Save(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.JSON(http.StatusCreated, user)
}

type Claims struct {
	UserID int `json:"userId"`
	jwt.RegisteredClaims
}

func (*Controller) JWTAuthMiddleware() gin.HandlerFunc {
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
				"message": "unauthorized",
			})
			c.Abort()
			return
		}

		userID := token.Claims.(*Claims).UserID

		_, err = client.User.Get(ctx, userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "not existing user id in token",
			})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}

var key *ecdsa.PrivateKey

func init() {
	var err error

	key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
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

func getCurrentUserID(c *gin.Context) int {
	userID, _ := c.Get("userID")
	return userID.(int)
}
