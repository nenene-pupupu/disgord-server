package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (*Controller) GetAllChats(c *gin.Context) {
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
