// Package condition contains helper functions for the k8s conditions
package condition

import (
	"context"
	"log/slog"
	"time"

	"gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/internal/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Log is an alias that implements slog.LogValuer.
type Log metav1.Condition

// SetStatusCondition sets the corresponding condition in conditions to newCondition and returns true
// if the conditions are changed by this call.
// conditions must be non-nil.
//  1. if the condition of the specified type already exists (all fields of the existing condition are updated to
//     newCondition, LastTransitionTime is set to now if the new status differs from the old status)
//  2. if a condition of the specified type does not exist (LastTransitionTime is set to now() if unset, and newCondition is appended)
//
// If  a logger is found in the context a status change is logged.
//
// Inspired by https://github.com/kubernetes/apimachinery/blob/master/pkg/api/meta/conditions.go
func SetStatusCondition(ctx context.Context, conditions *[]metav1.Condition, newCondition metav1.Condition) bool {
	if conditions == nil {
		return false
	}

	l := log.FromContext(ctx)

	existingCondition := FindStatusCondition(*conditions, newCondition.Type)

	if existingCondition == nil {
		l.Info("new condition", "condition", Log(newCondition))

		if newCondition.LastTransitionTime.IsZero() {
			newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		}

		*conditions = append(*conditions, newCondition)

		return true
	}

	changed := false
	origReason := existingCondition.Reason
	origStatus := existingCondition.Status

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		if !newCondition.LastTransitionTime.IsZero() {
			existingCondition.LastTransitionTime = newCondition.LastTransitionTime
		} else {
			existingCondition.LastTransitionTime = metav1.NewTime(time.Now())
		}

		changed = true
	}

	if existingCondition.Reason != newCondition.Reason {
		existingCondition.Reason = newCondition.Reason
		changed = true
	}

	if existingCondition.Message != newCondition.Message {
		existingCondition.Message = newCondition.Message
		changed = true
	}

	if existingCondition.ObservedGeneration != newCondition.ObservedGeneration {
		existingCondition.ObservedGeneration = newCondition.ObservedGeneration
		changed = true
	}

	if changed {
		l.Info("condition change",
			"type", newCondition.Type,
			"oldStatus", origStatus,
			"newStatus", newCondition.Status,
			"oldReason", origReason,
			"newReason", newCondition.Reason,
		)
	}

	return changed
}

// FindStatusCondition finds the conditionType in conditions.
//
// Inspired by https://github.com/kubernetes/apimachinery/blob/master/pkg/api/meta/conditions.go
func FindStatusCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}

	return nil
}

// HasConditionAndReason is similar to HasCondition but only returns true if reason matches too.
func HasConditionAndReason(conditions []metav1.Condition, c metav1.Condition) bool {
	found := FindStatusCondition(conditions, c.Type)
	if found == nil {
		return false
	}

	return found.Status == c.Status && c.Reason == found.Reason
}

// HasCondition will return true if the given
// conditions have a condition matching the provided
// Condition.
// Only the Type and Status field will be used in the comparison, meaning that
// this function will return 'true' even if the Reason, Message and
// LastTransitionTime fields do not match.
func HasCondition(conditions []metav1.Condition, c metav1.Condition) bool {
	found := FindStatusCondition(conditions, c.Type)
	if found == nil {
		return false
	}

	return found.Status == c.Status
}

// LogValue implements slog.LogValuer.
func (c Log) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("type", c.Type),
		slog.String("status", string(c.Status)),
		slog.String("reason", c.Reason),
		slog.String("message", c.Message),
	}
	if !c.LastTransitionTime.Time.IsZero() {
		attrs = append(attrs,
			slog.Time("lastTransition", c.LastTransitionTime.Time),
		)
	}

	if c.ObservedGeneration > 0 {
		attrs = append(attrs,
			slog.Int64("observedGeneration", c.ObservedGeneration),
		)
	}

	return slog.GroupValue(
		attrs...,
	)
}

// HasConditionFunc is a func that returns a func that checks if an object has wantedCondition condition.
// This can be useful in tests.
func HasConditionFunc(ctx context.Context, cli client.Client, obj client.Object, wantedCondition metav1.Condition) func() bool {
	return func() bool {
		l := log.FromContext(ctx)
		o, _ := obj.DeepCopyObject().(client.Object)

		key := client.ObjectKeyFromObject(obj)

		err := cli.Get(ctx, key, o)
		if err != nil {
			l.Error(err, "failed to get object", logger.Name(key)...)

			return false
		}

		conditions, ok, err := Conditions(o)
		if err != nil {
			l.Error(err, "failed to get conditions", logger.Name(key)...)
		}

		if !ok {
			l.Info("no conditions found", logger.Name(key)...)

			return false
		}

		return HasCondition(conditions, wantedCondition)
	}
}

// Conditions tries to get the conditions from an object. If it is not
// possible to get conditions the returned boolean is false.
func Conditions(obj client.Object) ([]metav1.Condition, bool, error) {
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return []metav1.Condition{}, false, err
	}

	return conditions(u)
}

// conditions tries to get the conditions from an unstructured map. If it is not
// possible to get conditions the returned boolean is false.
func conditions(u map[string]any) ([]metav1.Condition, bool, error) {
	items, ok, err := unstructured.NestedSlice(u, "status", "conditions")
	if err != nil || !ok {
		return []metav1.Condition{}, false, err
	}

	conditions := make([]metav1.Condition, 0, len(items))

	for i := range items {
		m, ok := items[i].(map[string]any)
		if !ok {
			continue
		}

		c := metav1.Condition{}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(m, &c); err != nil {
			continue
		}

		conditions = append(conditions, c)
	}

	return conditions, true, nil
}
