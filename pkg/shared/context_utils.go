package shared

import "golang.org/x/net/context"

type syncModeKey struct{}

func IsSyncMode(ctx context.Context) bool {
	v, _ := ctx.Value(syncModeKey{}).(bool)
	return v
}

// Only used in tests
func ContextWithSyncMode(ctx context.Context) context.Context {
	return context.WithValue(ctx, syncModeKey{}, true)
}
