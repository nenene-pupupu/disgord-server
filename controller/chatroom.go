package controller

import (
	"net/http"
	"strconv"

	"disgord/ent/chatroom"

	"github.com/gin-gonic/gin"
)

func (*Controller) GetAllChatrooms(c *gin.Context) {
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

func (*Controller) GetChatroomById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "failed to read id",
		})
		return
	}

	chatroom, err := client.Chatroom.
		Query().
		Where(chatroom.ID(id)).
		Only(ctx)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find chatroom",
		})
		return
	}

	c.JSON(http.StatusOK, chatroom)
}

func (*Controller) CreateChatroom(c *gin.Context) {
	var body struct {
		Name string
	}

	if err := c.Bind(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "failed to read body",
		})
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

	c.JSON(http.StatusOK, chatroom)
}

func (*Controller) UpdateChatroom(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "failed to read id",
		})
		return
	}

	var body struct {
		Name string
	}

	if err := c.Bind(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "failed to read body",
		})
		return
	}

	chatroom, err := client.Chatroom.
		UpdateOneID(id).
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
