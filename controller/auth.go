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

func (controller Controller) InitAuth() {
	group := controller.Router.Group("/auth")

	group.POST("/sign-in", func(c *gin.Context) {
		var auth Auth
		if err := c.Bind(&auth); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "failed to read body",
			})
			return
		}

		user, err := controller.Client.User.
			Query().
			Where(user.Username(auth.Username)).
			Only(controller.Ctx)
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
	})

	group.POST("/sign-up", func(c *gin.Context) {
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

		_, err := controller.Client.User.
			Query().
			Where(user.Username(body.Username)).
			Only(controller.Ctx)
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

		user, err := controller.Client.User.
			Create().
			SetUsername(body.Username).
			SetPassword(string(hash)).
			SetDisplayName(body.Username).
			Save(controller.Ctx)
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
	})

}
