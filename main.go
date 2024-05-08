//go:generate go run -mod=mod github.com/swaggo/swag/cmd/swag init

package main

import (
	"context"
	"log"

	"disgord/controller"
	_ "disgord/docs"
	"disgord/ent"

	"entgo.io/ent/dialect"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
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

	c := controller.NewController(ctx, client)

	auth := r.Group("/auth")
	{
		auth.POST("/sign-in", c.SignIn)
		auth.POST("/sign-up", c.SignUp)
	}

	user := r.Group("/user")
	{
		user.GET("", c.GetAllUsers)
		user.GET("/:id", c.GetUserById)
	}

	chatroom := r.Group("/chatroom")
	{
		chatroom.GET("", c.GetAllChatrooms)
		chatroom.GET("/:id", c.GetChatroomById)
		chatroom.POST("", c.CreateChatroom)
		chatroom.PATCH("/:id", c.UpdateChatroom)
	}

	chat := r.Group("/chat")
	{
		chat.GET("", c.GetAllChats)
	}

	ws := r.Group("/ws")
	{
		ws.GET("", c.GetWebsocket)
		// ws.GET("/:chatroomId", func(ctx *gin.Context) {})
		// ws.GET("/:chatroomId/voice", func(ctx *gin.Context) {})
		// ws.GET("/:chatroomId/video", func(ctx *gin.Context) {})
		go c.HandleBroadcast()
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Run()
}
