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
		field.Int("chatroom_id"),

		field.Int("sender_id"),

		field.String("content"),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Chat.
func (Chat) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("chatroom", Chatroom.Type).
			Ref("chats").
			Field("chatroom_id").
			Unique().
			Required(),

		edge.From("sender", User.Type).
			Ref("chats").
			Field("sender_id").
			Unique().
			Required(),
	}
}
