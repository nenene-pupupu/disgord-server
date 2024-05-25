package controller

import (
	"net/http"

	"disgord/ent/chat"

	"github.com/gin-gonic/gin"
)

// GetAllChats godoc
// @Tags	chat
// @Router	/chat [get]
// @Param	q query controller.GetAllChats.Query true "query"
func (*Controller) GetAllChats(c *gin.Context) {
	type Query struct {
		ChatroomID int `form:"chatroomId"`
		SenderID   int `form:"senderId"`
	}

	var query Query
	if err := c.BindQuery(&query); err != nil {
		return
	}

	chatQuery := client.Chat.Query()
	if query.ChatroomID != 0 {
		chatQuery = chatQuery.Where(chat.ChatroomID(query.ChatroomID))
	}
	if query.SenderID != 0 {
		chatQuery = chatQuery.Where(chat.SenderID(query.SenderID))
	}

	chats, err := chatQuery.All(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find chats",
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
