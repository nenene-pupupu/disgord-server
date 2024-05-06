package controller

import (
	"context"

	"disgord/ent"
)

type Controller struct{}

var (
	ctx    context.Context
	client *ent.Client
)

func NewController(_ctx context.Context, _client *ent.Client) *Controller {
	ctx = _ctx
	client = _client
	return &Controller{}
}
