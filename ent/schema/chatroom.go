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

		field.Int("owner_id"),

		field.Time("created_at").
			Default(time.Now()),
	}
}

// Edges of the Chatroom.
func (Chatroom) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("owner", User.Type).
			Field("owner_id").
			Unique().
			Required(),

		edge.To("chats", Chat.Type),
	}
}
