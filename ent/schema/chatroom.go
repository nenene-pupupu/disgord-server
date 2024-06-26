package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
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

		field.Bool("is_private").
			Default(false),

		field.String("password").
			Optional().
			Sensitive(),

		field.Int("owner_id"),

		field.Uint8("profile_color_index").
			Immutable(),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Chatroom.
func (Chatroom) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner", User.Type).
			Ref("chatrooms").
			Field("owner_id").
			Unique().
			Required(),

		edge.From("members", User.Type).
			Ref("allowed_chatrooms"),

		edge.To("chats", Chat.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
