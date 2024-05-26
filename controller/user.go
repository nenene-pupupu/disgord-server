package controller

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAllUsers godoc
//
//	@Tags		user
//	@Param		Authorization	header	string	true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	200	{array}	ent.User
//	@Failure	401	"unauthorized"
//	@Router		/user [get]
func (*Controller) GetAllUsers(c *gin.Context) {
	users, err := client.User.
		Query().
		All(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUserByID godoc
//
//	@Tags		user
//	@Param		uri				path	controller.GetUserByID.Uri	true	"path"
//	@Param		Authorization	header	string						true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	200	{object}	ent.User
//	@Failure	401	"unauthorized"
//	@Failure	404	"cannot find user"
//	@Router		/user/{id} [get]
func (*Controller) GetUserByID(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	user, err := client.User.Get(ctx, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser godoc
//
//	@Tags		user
//	@Param		uri				path	controller.DeleteUser.Uri	true	"path"
//	@Param		Authorization	header	string						true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	204
//	@Failure	401	"unauthorized"
//	@Failure	404	"cannot find user"
//	@Router		/user/{id} [delete]
func (*Controller) DeleteUser(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	err := client.User.
		DeleteOneID(uri.ID).
		Exec(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
