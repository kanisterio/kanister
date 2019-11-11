package field

import "context"

// context.Context support for fields

type contextKey uint8

const ctxKey = contextKey(43)

// FromContext returns the fields present in ctx if any
func FromContext(ctx context.Context) Fields {
	if ctx != nil {
		if s, ok := ctx.Value(ctxKey).(Fields); ok {
			return s
		}
	}
	return nil
}

// Context returns a new context that has ctx as its parent context and
// has a Field with the given key and value.
func Context(ctx context.Context, key string, v interface{}) context.Context {
	return context.WithValue(ctx, ctxKey, Add(FromContext(ctx), key, v))
}

// AddMapToContext returns a context that has ctx as its parent context and has
// fields populated from the keys and values in m.
func AddMapToContext(ctx context.Context, m M) context.Context {
	return context.WithValue(ctx, ctxKey, addMap(FromContext(ctx), m))
}
