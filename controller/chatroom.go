package controller

import (
	"net/http"

	"disgord/ent/chatroom"

	"github.com/gin-gonic/gin"
)

// GetAllChatrooms godoc
// @Tags	chatroom
// @Router	/chatroom [get]
// @Param	q query controller.GetAllChatrooms.Query true "query"
func (*Controller) GetAllChatrooms(c *gin.Context) {
	type Query struct {
		UserID int `form:"userId" binding:"omitempty"`
	}

	var query Query
	if err := c.BindQuery(&query); err != nil {
		return
	}

	chatrooms, err := client.Chatroom.
		Query().
		All(ctx)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, chatrooms)
}

// GetChatroomByID godoc
// @Tags	chatroom
// @Router	/chatroom/{id} [get]
// @Param	uri path controller.GetChatroomByID.Uri true "path"
func (*Controller) GetChatroomByID(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	chatroom, err := client.Chatroom.
		Query().
		Where(chatroom.ID(uri.ID)).
		Only(ctx)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find chatroom",
		})
		return
	}

	c.JSON(http.StatusOK, chatroom)
}

// CreateChatroom godoc
// @Tags	chatroom
// @Router	/chatroom [post]
// @Param	body body controller.CreateChatroom.Body true "body"
func (*Controller) CreateChatroom(c *gin.Context) {
	type Body struct {
		Name string `binding:"required"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		return
	}

	chatroom, err := client.Chatroom.
		Create().
		SetName(body.Name).
		Save(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, chatroom)
}

// UpdateChatroom godoc
// @Tags	chatroom
// @Router	/chatroom/{id} [patch]
// @Param	uri path controller.UpdateChatroom.Uri true "uri"
// @Param	body body controller.UpdateChatroom.Body true "body"
func (*Controller) UpdateChatroom(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	type Body struct {
		Name string `binding:"required"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		return
	}

	chatroom, err := client.Chatroom.
		UpdateOneID(uri.ID).
		SetName(body.Name).
		Save(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, chatroom)
}

// DeleteChatroom godoc
// @Tags	chatroom
// @Router	/chatroom/{id} [delete]
// @Param	uri path controller.DeleteChatroom.Uri true "uri"
func (*Controller) DeleteChatroom(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	err := client.Chatroom.
		DeleteOneID(uri.ID).
		Exec(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find chatroom",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
