package controller

import (
	"log"
	"net/http"

	"disgord/ent/user"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// SignIn godoc
// @Tags	Auth
// @Router	/auth/sign-in [post]
// @Param	auth body controller.Auth true "auth"
func (*Controller) SignIn(c *gin.Context) {
	var auth Auth
	if err := c.Bind(&auth); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "failed to read body",
		})
		return
	}

	user, err := client.User.
		Query().
		Where(user.Username(auth.Username)).
		Only(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "user not found",
		})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(auth.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "invalid username or password",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
	})
}

// SignUp godoc
// @Tags	Auth
// @Router	/auth/sign-up [post]
func (*Controller) SignUp(c *gin.Context) {
	var body struct {
		Username string
		Password string
	}

	if err := c.Bind(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "failed to read body",
		})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to hash password",
		})
		return
	}

	user, err := client.User.
		Create().
		SetUsername(body.Username).
		SetPassword(string(hash)).
		SetDisplayName(body.Username).
		Save(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to create user",
		})
		log.Print(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"name": user.Username,
	})
}
