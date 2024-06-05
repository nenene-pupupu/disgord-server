//go:generate go run -mod=mod github.com/swaggo/swag/cmd/swag init

package main

import (
	"disgord/controller"
	_ "disgord/docs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title disGOrd API
func main() {
	c := controller.New()
	defer c.Close()

	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173"}
	config.AllowCredentials = true
	config.AllowWebSockets = true
	config.AddAllowHeaders("Authorization")
	r.Use(cors.New(config))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	public := r.Group("")
	{
		auth := public.Group("/auth")
		{
			auth.POST("/sign-up", c.SignUp)
			auth.POST("/sign-in", c.SignIn)
			auth.POST("/refresh", c.Refresh)
			auth.POST("/sign-out", c.SignOut)
		}

		chatroom := public.Group("/chatrooms")
		{
			chatroom.GET("", c.GetAllChatrooms)
		}
	}

	private := r.Group("")
	private.Use(c.JWTAuthMiddleware())
	{
		user := private.Group("/users")
		{
			user.GET("", c.GetAllUsers)
			user.GET("/:id", c.GetUserByID)
			user.GET("/me", c.GetMyProfile)
			user.PATCH("/me", c.UpdateMyProfile)
			user.DELETE("/me", c.CancelAccount)
		}

		chatroom := private.Group("/chatrooms")
		{
			chatroom.GET("/:id", c.GetChatroomByID)
			chatroom.POST("", c.CreateChatroom)
			chatroom.PATCH("/:id", c.UpdateChatroom)
			chatroom.DELETE("/:id", c.DeleteChatroom)
			chatroom.POST("/:id/join", c.JoinChatroom)
			chatroom.PATCH("/:id/public", c.MakeChatroomPublic)
		}

		chat := private.Group("/chats")
		{
			chat.GET("", c.GetAllChats)
			chat.GET("/:id", c.GetChatByID)
		}

		ws := private.Group("/ws")
		{
			ws.GET("", c.ConnectWebsocket)
		}
	}

	r.Run()
}
