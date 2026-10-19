package identity

import (
	"context"

	"elimu-core/backend/internal/domain"
)

type actorContextKey struct{}

func WithActor(ctx context.Context, actor domain.User) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

func Actor(ctx context.Context) (domain.User, bool) {
	actor, ok := ctx.Value(actorContextKey{}).(domain.User)
	return actor, ok
}
