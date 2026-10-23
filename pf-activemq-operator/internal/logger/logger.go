// Package logger contains logger functions
//
//nolint:revive // no need to document every function here
package logger

import (
	"log/slog"

	"gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/internal/constant"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Type(typeValue string) slog.Attr {
	return slog.Any(constant.LogKeyType, typeValue)
}

func Reason(reason string) slog.Attr {
	return slog.Any(constant.LogKeyReason, reason)
}

func Name(key client.ObjectKey) []any {
	return []any{constant.LogKeyName, key}
}
