package controller

import (
	"context"

	"disgord/ent"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	Router *gin.Engine
	Ctx    context.Context
	Client *ent.Client
}

func (controller Controller) Init() {
	controller.InitAuth()
	controller.InitUser()
	controller.InitChat()
}
