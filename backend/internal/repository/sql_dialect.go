package repository

import (
	"entgo.io/ent/dialect"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

func isPostgresEntClient(client *dbent.Client) bool {
	return client != nil && client.Driver().Dialect() == dialect.Postgres
}
