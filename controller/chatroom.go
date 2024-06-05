package controller

import (
	"log"
	"net/http"

	"disgord/ent/chatroom"
	"disgord/ent/user"

	"github.com/gin-gonic/gin"
)

// GetAllChatrooms godoc
//
//	@Tags		chatroom
//	@Summary	list all chatrooms with the given query
//	@Param		q	query	controller.GetAllChatrooms.Query	true	"query"
//	@Success	200	{array}	ent.Chatroom
//	@Router		/chatrooms [get]
func (*Controller) GetAllChatrooms(c *gin.Context) {
	type Query struct {
		OwnerID  int `form:"ownerId"`
		MemberID int `form:"memberId"`
	}

	var query Query
	if err := c.BindQuery(&query); err != nil {
		return
	}

	chatroomQuery := client.Chatroom.Query()
	if query.OwnerID != 0 {
		chatroomQuery = chatroomQuery.Where(chatroom.OwnerID(query.OwnerID))
	}
	if query.MemberID != 0 {
		chatroomQuery = chatroomQuery.Where(chatroom.HasMembersWith(user.ID(query.MemberID)))
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
//	@Failure	401
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
//	@Failure	401
//	@Failure	404	"cannot find user"
//	@Router		/chatrooms [post]
func (*Controller) CreateChatroom(c *gin.Context) {
	type Body struct {
		Name     string `json:"name" binding:"required"`
		Password string `json:"password"`
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

	user, err := tx.User.Get(ctx, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	chatroomCreate := tx.Chatroom.
		Create().
		SetName(body.Name).
		SetOwnerID(userID).
		SetProfileColorIndex(user.ProfileColorIndex)
	if body.Password != "" {
		chatroomCreate = chatroomCreate.
			SetIsPrivate(true).
			SetPassword(hashPassword(body.Password)).
			AddMemberIDs(userID)
	}

	chatroom, err := chatroomCreate.Save(ctx)
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

	c.JSON(http.StatusCreated, chatroom)
}

// UpdateChatroom godoc
//
//	@Description	If password is not provided, it will be public, i.e. clear the password and the member list.
//	@Tags			chatroom
//	@Summary		update the chatroom
//	@Param			uri				path	controller.UpdateChatroom.Uri	true	"uri"
//	@Param			Authorization	header	string							true	"Bearer AccessToken"
//	@Param			body			body	controller.UpdateChatroom.Body	false	"Request body"
//	@Security		BearerAuth
//	@Success		200	{object}	ent.Chatroom
//	@Failure		401
//	@Failure		403	"chatroom owner only"
//	@Failure		404	"cannot find chatroom"
//	@Router			/chatrooms/{id} [patch]
func (*Controller) UpdateChatroom(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	type Body struct {
		Name     string `json:"name"`
		Password string `json:"password"`
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

	chatroomUpdate := chatroom.Update()
	if body.Name != "" {
		chatroomUpdate = chatroomUpdate.SetName(body.Name)
	}
	if body.Password != "" {
		chatroomUpdate = chatroomUpdate.SetPassword(hashPassword(body.Password))

		if !chatroom.IsPrivate {
			chatroomUpdate = chatroomUpdate.
				SetIsPrivate(true).
				AddMemberIDs(userID)
		}
	} else {
		chatroomUpdate = chatroomUpdate.
			SetIsPrivate(false).
			ClearPassword().
			ClearMembers()
	}

	chatroom, err = chatroomUpdate.Save(ctx)
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
//	@Failure	401
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

// JoinChatroom godoc
//
//	@Description	If the chatroom is public or the user is already a member of the private chatroom, it will ignore the password.
//	@Description	Otherwise, the user must provide the password to join.
//	@Tags			chatroom
//	@Summary		join the chatroom, with password if it is private
//	@Param			uri				path	controller.JoinChatroom.Uri		true	"uri"
//	@Param			Authorization	header	string							true	"Bearer AccessToken"
//	@Param			body			body	controller.JoinChatroom.Body	true	"Request body"
//	@Security		BearerAuth
//	@Success		200
//	@Failure		401
//	@Failure		403	"not a member of the chatroom, password required"
//	@Failure		404	"cannot find chatroom"
//	@Router			/chatrooms/{id}/join [post]
func (*Controller) JoinChatroom(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	type Body struct {
		Muted    *bool  `json:"muted" binding:"required"`
		CamOn    *bool  `json:"camOn" binding:"required"`
		Password string `json:"password"`
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

	chatroom, err := tx.Chatroom.Get(ctx, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find chatroom",
		})
		return
	}

	if chatroom.IsPrivate {
		_, err = chatroom.QueryMembers().
			Where(user.ID(userID)).
			Only(ctx)
		if err != nil {
			if body.Password == "" {
				c.JSON(http.StatusForbidden, gin.H{
					"message": "not a member of the chatroom, password required",
				})
				return
			}

			if !verifyPassword(chatroom.Password, body.Password) {
				c.JSON(http.StatusForbidden, gin.H{
					"message": "incorrect password",
				})
				return
			}

			_, err = chatroom.Update().
				AddMemberIDs(userID).
				Save(ctx)
			if err != nil {
				c.Status(http.StatusInternalServerError)
				log.Println(err)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	joinRoom(uri.ID, userID, *body.Muted, *body.CamOn)

	c.Status(http.StatusOK)
}
