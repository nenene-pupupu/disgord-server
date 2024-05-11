package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Chat holds the schema definition for the Chat entity.
type Chat struct {
	ent.Schema
}

// Fields of the Chat.
func (Chat) Fields() []ent.Field {
	return []ent.Field{
		field.String("content"),

		field.Time("created_at").
			Optional().
			Default(time.Now),
	}
}

// Edges of the Chat.
func (Chat) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("chatroom", Chatroom.Type).
			Ref("chats").
			Unique(),

		edge.From("sender", User.Type).
			Ref("chats").
			Unique(),
	}
}
