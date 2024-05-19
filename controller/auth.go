package controller

import (
	"net/http"

	"disgord/ent/user"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// SignIn godoc
// @Tags	auth
// @Router	/auth/sign-in [post]
// @Param	body body controller.SignIn.Body true "body"
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

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))
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
// @Tags	auth
// @Router	/auth/sign-up [post]
// @Param	body body controller.SignUp.Body true "body"
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
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"name": user.Username,
	})
}
