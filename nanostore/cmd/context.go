package main

import "context"

type contextKey string

const queryContextKey = contextKey("query")

// withQuery returns a new context with the given Query object.
func withQuery(ctx context.Context, query *Query) context.Context {
	return context.WithValue(ctx, queryContextKey, query)
}

// fromContext returns the Query object from the given context.
func fromContext(ctx context.Context) (*Query, bool) {
	query, ok := ctx.Value(queryContextKey).(*Query)
	return query, ok
}
