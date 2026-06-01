package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentGroupDelegation holds exclusive group repricing delegated from a manager to a direct child.
type AgentGroupDelegation struct {
	ent.Schema
}

func (AgentGroupDelegation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "agent_group_delegations"},
	}
}

func (AgentGroupDelegation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (AgentGroupDelegation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("manager_user_id"),
		field.Int64("child_user_id"),
		field.Int64("group_id"),
		field.Float("rate_multiplier").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}),
		field.Bool("can_delegate").
			Default(false),
	}
}

func (AgentGroupDelegation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("manager", User.Type).
			Ref("managed_group_delegations").
			Field("manager_user_id").
			Unique().
			Required(),
		edge.From("child", User.Type).
			Ref("received_group_delegations").
			Field("child_user_id").
			Unique().
			Required(),
		edge.From("group", Group.Type).
			Ref("agent_group_delegations").
			Field("group_id").
			Unique().
			Required(),
	}
}

func (AgentGroupDelegation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("manager_user_id", "child_user_id", "group_id").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("child_user_id"),
		index.Fields("manager_user_id"),
		index.Fields("group_id"),
	}
}
