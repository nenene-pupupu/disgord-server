package controller

import (
	"net/http"

	"disgord/ent/user"

	"github.com/gin-gonic/gin"
)

type UserDao struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// GetAllUsers godoc
// @Tags	user
// @Router	/user [get]
func (*Controller) GetAllUsers(c *gin.Context) {
	var users []UserDao
	err := client.User.Query().
		Select(user.FieldID).
		Select(user.FieldUsername).
		Select(user.FieldDisplayName).
		Scan(ctx, &users)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUserByID godoc
// @Tags	user
// @Router	/user/{id} [get]
// @Param	uri path controller.GetUserByID.Uri true "path"
func (*Controller) GetUserByID(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	var users []UserDao
	err := client.User.
		Query().
		Where(user.ID(uri.ID)).
		Select(user.FieldID).
		Select(user.FieldUsername).
		Select(user.FieldDisplayName).
		Scan(ctx, &users)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	c.JSON(http.StatusOK, users[0])
}
