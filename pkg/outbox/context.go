package outbox

import (
	"context"
)

type settingsKey struct{}

// ContextSettings are settings that can configure outbox behaviour through context
type ContextSettings struct {
	Namespace string
}

// Clone clones context settings
func (c ContextSettings) Clone() *ContextSettings {
	return &c
}

func settingsFromContext(ctx context.Context) *ContextSettings {
	return ctx.Value(settingsKey{}).(*ContextSettings)
}

func contextWithSettings(ctx context.Context, newCtx ContextSettings) context.Context {
	return context.WithValue(ctx, settingsKey{}, &newCtx)
}

func augmentContextSettings(ctx context.Context, f func(c *ContextSettings)) context.Context {
	c := settingsFromContext(ctx)
	if c == nil {
		c = &ContextSettings{}
	}

	c = c.Clone()
	f(c)

	return contextWithSettings(ctx, *c)
}

// NamespaceFromContext identifies what namespace to record published messages to in the outbox
func NamespaceFromContext(ctx context.Context) string {
	c := settingsFromContext(ctx)
	if c == nil {
		return ""
	}

	return c.Namespace
}

// WithNamespace creates a context which configures published messages to be recorded to the outbox with the specified
// namespace
func WithNamespace(ctx context.Context, namespace string) context.Context {
	return augmentContextSettings(ctx, func(c *ContextSettings) {
		c.Namespace = namespace
	})
}
