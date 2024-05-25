package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAllChats godoc
// @Tags	chat
// @Router	/chat [get]
// @Param	q query controller.GetAllChats.Query true "query"
func (*Controller) GetAllChats(c *gin.Context) {
	type Query struct {
		ChatroomID int `form:"chatroomId"`
		UserID     int `form:"userId"`
	}

	var query Query
	if err := c.BindQuery(&query); err != nil {
		return
	}

	chats, err := client.Chat.
		Query().
		All(ctx)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, chats)
}

// GetChatByID godoc
// @Tags	chat
// @Router	/chat/{id} [get]
// @Param	uri path controller.GetChatByID.Uri true "path"
func (*Controller) GetChatByID(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	chat, err := client.Chat.Get(ctx, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find chat",
		})
		return
	}

	c.JSON(http.StatusOK, chat)
}
