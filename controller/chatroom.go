package controller

import (
	"log"
	"net/http"

	"disgord/ent/chatroom"

	"github.com/gin-gonic/gin"
)

// GetAllChatrooms godoc
//
//	@Tags		chatroom
//	@Summary	list all chatrooms with the given query
//	@Param		q				query	controller.GetAllChatrooms.Query	true	"query"
//	@Param		Authorization	header	string								true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	200	{array}	ent.Chatroom
//	@Failure	401	"unauthorized"
//	@Router		/chatrooms [get]
func (*Controller) GetAllChatrooms(c *gin.Context) {
	type Query struct {
		OwnerID int `form:"ownerId" binding:"omitempty"`
	}

	var query Query
	if err := c.BindQuery(&query); err != nil {
		return
	}

	chatroomQuery := client.Chatroom.Query()
	if query.OwnerID != 0 {
		chatroomQuery = chatroomQuery.Where(chatroom.OwnerID(query.OwnerID))
	}

	chatrooms, err := chatroomQuery.All(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.JSON(http.StatusOK, chatrooms)
}

// GetChatroomByID godoc
//
//	@Tags		chatroom
//	@Summary	get a single chatroom by id
//	@Param		uri				path	controller.GetChatroomByID.Uri	true	"path"
//	@Param		Authorization	header	string							true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	200	{object}	ent.Chatroom
//	@Failure	401	"unauthorized"
//	@Failure	404	"cannot find chatroom"
//	@Router		/chatrooms/{id} [get]
func (*Controller) GetChatroomByID(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	chatroom, err := client.Chatroom.Get(ctx, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find chatroom",
		})
		return
	}

	c.JSON(http.StatusOK, chatroom)
}

// CreateChatroom godoc
//
//	@Tags		chatroom
//	@Summary	create a new chatroom
//	@Param		Authorization	header	string							true	"Bearer AccessToken"
//	@Param		body			body	controller.CreateChatroom.Body	true	"Request body"
//	@Security	BearerAuth
//	@Success	201	{object}	ent.Chatroom
//	@Failure	401	"unauthorized"
//	@Router		/chatrooms [post]
func (*Controller) CreateChatroom(c *gin.Context) {
	type Body struct {
		Name     string `binding:"required"`
		Password string `binding:"omitempty"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		return
	}

	userID := getCurrentUserID(c)

	chatroomCreate := client.Chatroom.
		Create().
		SetName(body.Name).
		SetOwnerID(userID)
	if body.Password != "" {
		chatroomCreate.SetPassword(body.Password)
	}

	chatroom, err := chatroomCreate.Save(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.JSON(http.StatusCreated, chatroom)
}

// UpdateChatroom godoc
//
//	@Tags		chatroom
//	@Summary	update the chatroom
//	@Param		uri				path	controller.UpdateChatroom.Uri	true	"uri"
//	@Param		Authorization	header	string							true	"Bearer AccessToken"
//	@Param		body			body	controller.UpdateChatroom.Body	false	"Request body"
//	@Security	BearerAuth
//	@Success	200	{object}	ent.Chatroom
//	@Failure	401	"unauthorized"
//	@Failure	404	"cannot find chatroom"
//	@Router		/chatrooms/{id} [patch]
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
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find chatroom",
		})
		return
	}

	c.JSON(http.StatusOK, chatroom)
}

// DeleteChatroom godoc
//
//	@Tags		chatroom
//	@Summary	delete the chatroom and all chats in it
//	@Param		uri				path	controller.DeleteChatroom.Uri	true	"uri"
//	@Param		Authorization	header	string							true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	204
//	@Failure	401	"unauthorized"
//	@Failure	403	"chatroom owner only"
//	@Failure	404	"cannot find chatroom"
//	@Router		/chatrooms/{id} [delete]
func (*Controller) DeleteChatroom(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	userID := getCurrentUserID(c)

	tx, err := client.Tx(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer tx.Rollback()

	chatroom, err := tx.Chatroom.Get(ctx, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find chatroom",
		})
		return
	}

	if chatroom.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "chatroom owner only",
		})
		return
	}

	err = tx.Chatroom.
		DeleteOneID(uri.ID).
		Exec(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find chatroom",
		})
		return
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	kickAllClientsFromRoom(uri.ID)

	c.Status(http.StatusNoContent)
}
