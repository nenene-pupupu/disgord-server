package controller

import (
	"net/http"
	"strconv"

	"disgord/ent/user"

	"github.com/gin-gonic/gin"
)

func (controller Controller) InitUser() {
	group := controller.Router.Group("/user")

	group.GET("", func(c *gin.Context) {
		users, err := controller.Client.User.
			Query().
			Select(user.FieldID).
			Select(user.FieldUsername).
			All(controller.Ctx)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, users)
	})

	group.GET("/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "failed to read id",
			})
			return
		}

		user, err := controller.Client.User.
			Query().
			Where(user.ID(id)).
			Select(user.FieldID).
			Select(user.FieldUsername).
			Only(controller.Ctx)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "cannot find user",
			})
			return
		}

		c.JSON(http.StatusOK, user)
	})
}
