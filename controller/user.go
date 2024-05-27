package controller

import (
	"log"
	"net/http"

	"disgord/ent/auth"
	"disgord/ent/user"
	"disgord/jwt"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// GetAllUsers godoc
//
//	@Tags		user
//	@Param		Authorization	header	string	true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	200	{array}	ent.User
//	@Failure	401	"unauthorized"
//	@Router		/user [get]
func (*Controller) GetAllUsers(c *gin.Context) {
	users, err := client.User.
		Query().
		All(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUserByID godoc
//
//	@Tags		user
//	@Param		uri				path	controller.GetUserByID.Uri	true	"path"
//	@Param		Authorization	header	string						true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	200	{object}	ent.User
//	@Failure	401	"unauthorized"
//	@Failure	404	"cannot find user"
//	@Router		/user/{id} [get]
func (*Controller) GetUserByID(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	user, err := client.User.Get(ctx, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetMyProfile godoc
//
//	@Tags		user
//	@Param		Authorization	header	string	true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	200	{object}	ent.User
//	@Failure	401	"unauthorized"
//	@Failure	404	"cannot find user"
//	@Router		/user/me [get]
func (*Controller) GetMyProfile(c *gin.Context) {
	userID, ok := jwt.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "unauthorized",
		})
		return
	}

	user, err := client.User.Get(ctx, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser godoc
//
//	@Tags		user
//	@Param		uri				path	controller.UpdateUser.Uri	true	"path"
//	@Param		Authorization	header	string						true	"Bearer AccessToken"
//	@Param		body			body	controller.UpdateUser.Body	true	"Request body"
//	@Security	BearerAuth
//	@Success	200	{object}	ent.User
//	@Failure	401	"unauthorized"
//	@Failure	403	"user can only update itself"
//	@Failure	404	"cannot find user"
//	@Router		/user/{id} [patch]
func (*Controller) UpdateUser(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	type Body struct {
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		return
	}

	userID, ok := jwt.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "unauthorized",
		})
		return
	}

	if userID != uri.ID {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "user can only update itself",
		})
		return
	}

	tx, err := client.Tx(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer tx.Rollback()

	_, err = tx.User.Get(ctx, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	if body.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			log.Println(err)
			return
		}

		_, err = tx.Auth.
			Update().
			Where(auth.HasUserWith(user.ID(uri.ID))).
			SetPassword(string(hash)).
			Save(ctx)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			log.Println(err)
			return
		}
	}

	if body.DisplayName != "" {
		_, err := tx.User.
			UpdateOneID(uri.ID).
			SetDisplayName(body.DisplayName).
			Save(ctx)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			log.Println(err)
			return
		}
	}

	user, err := tx.User.Get(ctx, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser godoc
//
//	@Tags		user
//	@Param		uri				path	controller.DeleteUser.Uri	true	"path"
//	@Param		Authorization	header	string						true	"Bearer AccessToken"
//	@Param		body			body	controller.DeleteUser.Body	true	"Request body"
//	@Security	BearerAuth
//	@Success	204
//	@Failure	401	"invalid password"
//	@Failure	403	"user can only cancel account itself"
//	@Failure	404	"cannot find user"
//	@Router		/user/{id} [delete]
func (*Controller) DeleteUser(c *gin.Context) {
	type Uri struct {
		ID int `uri:"id" binding:"required"`
	}

	var uri Uri
	if err := c.BindUri(&uri); err != nil {
		return
	}

	type Body struct {
		Password string `json:"password" binding:"required"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		return
	}

	userID, ok := jwt.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "unauthorized",
		})
		return
	}

	if userID != uri.ID {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "user can only cancel account itself",
		})
		return
	}

	tx, err := client.Tx(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer tx.Rollback()

	auth, err := tx.Auth.
		Query().
		Where(auth.HasUserWith(user.ID(uri.ID))).
		Only(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(auth.Password), []byte(body.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "invalid password",
		})
		return
	}

	err = tx.User.
		DeleteOneID(uri.ID).
		Exec(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.Status(http.StatusNoContent)
}
