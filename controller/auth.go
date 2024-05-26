package controller

import (
	"log"
	"net/http"

	"disgord/ent/auth"
	"disgord/ent/user"
	"disgord/jwt"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type Token struct {
	AccessToken string `json:"accessToken"`
}

// SignIn godoc
//
//	@Description	Sign in and receive an access token.
//	@Description	Set "Authorization" header with the "Bearer ${accessToken}" to authenticate requests.
//	@Tags			auth
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

	tokenString, err := jwt.IssueToken(auth.UserID)
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
//	@Param		body	body		controller.SignUp.Body	true	"Request body"
//	@Success	201		{object}	ent.User
//	@Failure	409		"username already exists"
//	@Router		/auth/sign-up [post]
func (*Controller) SignUp(c *gin.Context) {
	type Body struct {
		Username string `binding:"required"`
		Password string `binding:"required"`
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
		SetDisplayName(body.Username).
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
