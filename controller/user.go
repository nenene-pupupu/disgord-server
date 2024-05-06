package controller

import (
	"net/http"
	"strconv"

	"disgord/ent/user"

	"github.com/gin-gonic/gin"
)

func (*Controller) GetAllUsers(c *gin.Context) {
	users, err := client.User.
		Query().
		Select(user.FieldID).
		Select(user.FieldUsername).
		All(ctx)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (*Controller) GetUserById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "failed to read id",
		})
		return
	}

	user, err := client.User.
		Query().
		Where(user.ID(id)).
		Select(user.FieldID).
		Select(user.FieldUsername).
		Only(ctx)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}
