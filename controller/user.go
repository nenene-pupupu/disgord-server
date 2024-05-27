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
//	@Summary	list all users
//	@Param		Authorization	header	string	true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	200	{array}	ent.User
//	@Failure	401	"unauthorized"
//	@Router		/users [get]
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
//	@Summary	get a single user by id
//	@Param		uri				path	controller.GetUserByID.Uri	true	"path"
//	@Param		Authorization	header	string						true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	200	{object}	ent.User
//	@Failure	401	"unauthorized"
//	@Failure	404	"cannot find user"
//	@Router		/users/{id} [get]
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
//	@Summary	get the current user
//	@Param		Authorization	header	string	true	"Bearer AccessToken"
//	@Security	BearerAuth
//	@Success	200	{object}	ent.User
//	@Failure	401	"unauthorized"
//	@Failure	404	"cannot find user"
//	@Router		/users/me [get]
func (*Controller) GetMyProfile(c *gin.Context) {
	userID := jwt.GetCurrentUserID(c)

	user, err := client.User.Get(ctx, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateMyProfile godoc
//
//	@Tags		user
//	@Summary	update the current user
//	@Param		Authorization	header	string							true	"Bearer AccessToken"
//	@Param		body			body	controller.UpdateMyProfile.Body	true	"Request body"
//	@Security	BearerAuth
//	@Success	200	{object}	ent.User
//	@Failure	401	"unauthorized"
//	@Failure	404	"cannot find user"
//	@Router		/users/me [patch]
func (*Controller) UpdateMyProfile(c *gin.Context) {
	type Body struct {
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		return
	}

	userID := jwt.GetCurrentUserID(c)

	tx, err := client.Tx(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer tx.Rollback()

	if body.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			log.Println(err)
			return
		}

		_, err = tx.Auth.
			Update().
			Where(auth.HasUserWith(user.ID(userID))).
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
			UpdateOneID(userID).
			SetDisplayName(body.DisplayName).
			Save(ctx)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			log.Println(err)
			return
		}
	}

	user, err := tx.User.Get(ctx, userID)
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

// CancelAccount godoc
//
//	@Tags		user
//	@Summary	cancel the current user account and delete all related data
//	@Param		Authorization	header	string							true	"Bearer AccessToken"
//	@Param		body			body	controller.CancelAccount.Body	true	"Request body"
//	@Security	BearerAuth
//	@Success	204
//	@Failure	401	"invalid password"
//	@Failure	404	"cannot find user"
//	@Router		/users/me [delete]
func (*Controller) CancelAccount(c *gin.Context) {
	type Body struct {
		Password string `json:"password" binding:"required"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		return
	}

	userID := jwt.GetCurrentUserID(c)

	tx, err := client.Tx(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer tx.Rollback()

	auth, err := tx.Auth.
		Query().
		Where(auth.HasUserWith(user.ID(userID))).
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
		DeleteOneID(userID).
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
