package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").
			Unique(),

		field.String("password").
			Sensitive(),

		field.String("refresh_token").
			Optional().
			Sensitive(),

		field.String("display_name"),

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

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("chatrooms", Chatroom.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),

		edge.To("allowed_chatrooms", Chatroom.Type),

		edge.To("chats", Chat.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
