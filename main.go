//go:generate go run -mod=mod github.com/swaggo/swag/cmd/swag init

package main

import (
	"context"
	"log"

	"disgord/controller"
	_ "disgord/docs"
	"disgord/ent"
	"disgord/jwt"

	"entgo.io/ent/dialect"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title disGOrd API
func main() {
	client, err := ent.Open(dialect.SQLite, "file:disgord.db?cache=shared&_fk=1")
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Run the automatic migration tool to create all schema resources.
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowWebSockets = true
	config.AddAllowHeaders("Authorization")
	r.Use(cors.New(config))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	c := controller.NewController(ctx, client)

	auth := r.Group("/auth")
	{
		auth.POST("/sign-in", c.SignIn)
		auth.POST("/sign-up", c.SignUp)
	}

	r.Use(jwt.JWTAuthMiddleware())

	user := r.Group("/user")
	{
		user.GET("", c.GetAllUsers)
		user.GET("/:id", c.GetUserByID)
		user.GET("/me", c.GetMyProfile)
		user.PATCH("/:id", c.UpdateUser)
		user.DELETE("/:id", c.DeleteUser)
	}

	chatroom := r.Group("/chatroom")
	{
		chatroom.GET("", c.GetAllChatrooms)
		chatroom.GET("/:id", c.GetChatroomByID)
		chatroom.POST("", c.CreateChatroom)
		chatroom.PATCH("/:id", c.UpdateChatroom)
		chatroom.DELETE("/:id", c.DeleteChatroom)
	}

	chat := r.Group("/chat")
	{
		chat.GET("", c.GetAllChats)
		chat.GET("/:id", c.GetChatByID)
	}

	ws := r.Group("/ws")
	{
		ws.GET("", c.ConnectWebsocket)
	}

	r.Run()
}
