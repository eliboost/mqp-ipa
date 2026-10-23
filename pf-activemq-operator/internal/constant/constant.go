// Package constant contains constants definitions
//
//nolint:revive // this package does not require comment on every line
package constant

// log keys

const (
	LogKeyType   = "type"
	LogKeyReason = "reason"
	LogKeyName   = "name"
)

// ContextKey is used for context keys
type ContextKey string
