package controller

import (
	"log"
	"net/http"

	"disgord/ent"
	"disgord/ent/chat"
	"disgord/ent/user"

	"github.com/gin-gonic/gin"
)

// GetAllChats godoc
//
//	@Tags		chat
//	@Summary	list all chats with the given query
//	@Param		q				query	controller.GetAllChats.Query	true	"query"
//	@Param		Authorization	header	string							true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	200	{array}	controller.GetAllChats.Response
//	@Failure	401	"unauthorized"
//	@Router		/chats [get]
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
//	@Failure	401	"unauthorized"
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
