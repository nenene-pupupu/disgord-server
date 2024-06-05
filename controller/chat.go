package controller

import (
	"log"
	"net/http"

	"disgord/ent"
	"disgord/ent/chat"
	"disgord/ent/user"

	"entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
)

// GetAllChats godoc
//
//	@Description	It supports latest-first paging by offset and limit, and returns in oldest-first order.
//	@Tags			chat
//	@Summary		list all chats with the given query
//	@Param			q				query	controller.GetAllChats.Query	true	"query"
//	@Param			Authorization	header	string							true	"Bearer AccessToken"
//	@Security		BearerAuth
//	@Success		200	{array}	controller.GetAllChats.Response
//	@Failure		401
//	@Router			/chats [get]
func (*Controller) GetAllChats(c *gin.Context) {
	type Query struct {
		ChatroomID int `form:"chatroomId"`
		SenderID   int `form:"senderId"`
		Offset     int `form:"offset"`
		Limit      int `form:"limit"`
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

	chatQuery = chatQuery.Order(chat.ByCreatedAt(sql.OrderDesc()))
	if query.Offset != 0 {
		chatQuery = chatQuery.Offset(query.Offset)
	}
	if query.Limit != 0 {
		chatQuery = chatQuery.Limit(query.Limit)
	}

	chats, err := chatQuery.
		WithSender(func(uq *ent.UserQuery) {
			uq.Select(user.FieldDisplayName, user.FieldProfileColorIndex)
		}).
		All(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	for i, j := 0, len(chats)-1; i < j; i, j = i+1, j-1 {
		chats[i], chats[j] = chats[j], chats[i]
	}

	type Response struct {
		*ent.Chat
		Name  string `json:"displayName"`
		Color uint8  `json:"profileColorIndex"`
	}

	response := make([]Response, 0, len(chats))
	for _, chat := range chats {
		response = append(response, Response{
			Chat:  chat,
			Name:  chat.Edges.Sender.DisplayName,
			Color: chat.Edges.Sender.ProfileColorIndex,
		})
	}

	c.JSON(http.StatusOK, response)
}

// GetChatByID godoc
//
//	@Tags		chat
//	@Summary	get a single chat by id
//	@Param		uri				path	controller.GetChatByID.Uri	true	"path"
//	@Param		Authorization	header	string						true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	200	{object}	ent.Chat
//	@Failure	401
//	@Failure	404	"cannot find chat"
//	@Router		/chats/{id} [get]
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

// CreateChat godoc
//
//	@Tags		chat
//	@Summary	create a new chat
//	@Param		Authorization	header	string						true	"Bearer AccessToken"
//	@Param		body			body	controller.CreateChat.Body	true	"Request body"
//	@Security	BearerAuth
//	@Success	201	{object}	ent.Chat
//	@Failure	401
//	@Router		/chats [post]
func (*Controller) CreateChat(c *gin.Context) {
	type Body struct {
		ChatroomID int    `json:"chatroomId" binding:"required"`
		SenderID   int    `json:"senderId" binding:"required"`
		Content    string `json:"content" binding:"required"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		return
	}

	chat, err := client.Chat.
		Create().
		SetChatroomID(body.ChatroomID).
		SetSenderID(body.SenderID).
		SetContent(body.Content).
		Save(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.JSON(http.StatusCreated, chat)
}

// UpdateChat godoc
//
//	@Tags		chat
//	@Summary	update the chat
//	@Param		uri				path	controller.UpdateChat.Uri	true	"path"
//	@Param		Authorization	header	string						true	"Bearer AccessToken"
//	@Param		body			body	controller.UpdateChat.Body	false	"Request body"
//	@Security	BearerAuth
//	@Success	200	{object}	ent.Chat
//	@Failure	401
//	@Failure	403	"chat sender only"
//	@Failure	404	"cannot find chat"
//	@Router		/chats/{id} [patch]
func (*Controller) UpdateChat(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	type Body struct {
		Content string `json:"content"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
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

	chat, err := tx.Chat.Get(ctx, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find chat",
		})
		return
	}

	if chat.SenderID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "chat sender only",
		})
		return
	}

	chatUpdate := chat.Update()
	if body.Content != "" {
		chatUpdate = chatUpdate.SetContent(body.Content)
	}

	chat, err = chatUpdate.Save(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.JSON(http.StatusOK, chat)
}

// DeleteChat godoc
//
//	@Tags		chat
//	@Summary	delete the chat
//	@Param		uri				path	controller.DeleteChat.Uri	true	"path"
//	@Param		Authorization	header	string						true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	204
//	@Failure	401
//	@Failure	403	"chat sender only"
//	@Failure	404	"cannot find chat"
//	@Router		/chats/{id} [delete]
func (*Controller) DeleteChat(c *gin.Context) {
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

	chat, err := tx.Chat.Get(ctx, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find chat",
		})
		return
	}

	if chat.SenderID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "chat sender only",
		})
		return
	}

	err = tx.Chat.
		DeleteOneID(uri.ID).
		Exec(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.Status(http.StatusNoContent)
}
