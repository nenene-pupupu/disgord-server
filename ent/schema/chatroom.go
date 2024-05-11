package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Chatroom holds the schema definition for the Chatroom entity.
type Chatroom struct {
	ent.Schema
}

// Fields of the Chatroom.
func (Chatroom) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),

		field.String("password").
			Optional(),

		field.Time("created_at").
			Default(time.Now()),
	}
}

// Edges of the Chatroom.
func (Chatroom) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("owner", User.Type).
			Unique(),

		edge.To("chats", Chat.Type),

		// edge.To("members", User.Type),
	}
}
