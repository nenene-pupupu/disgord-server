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
