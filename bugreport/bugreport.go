// Package bugreport emits ticket-like bug reports as structured log messages.
//
// A bug report is a single slog record with a fixed message and a "bug_report"
// group holding standardized fields. Downstream infrastructure (e.g. a GCP log
// sink filtering on jsonPayload.bug_report.id) picks these records up and files
// them in an issue tracker. This package only owns the log shape; it knows
// nothing about the tracker.
//
// Example:
//
//	const (
//		Org         bugreport.Key = "org_id"
//		Integration bugreport.Key = "integration_id"
//	)
//
//	// In middleware, as identifiers become known:
//	ctx = bugreport.WithAttribute(ctx, Org, orgID)
//
//	// Later, anywhere downstream:
//	id, err := bugreport.Submit(ctx, bugreport.Report{
//		Title:       "Sync stalls after token refresh",
//		Description: "Reads stop advancing once the OAuth token is refreshed.",
//		Priority:    bugreport.High,
//		Reporter:    "jane@example.com",
//		Labels:      []string{"sync", "oauth"},
//		Subject:     bugreport.Subject{Integration: integrationID},
//	})
package bugreport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/amp-labs/amp-common/logger"
	"github.com/google/uuid"
)

// Message is the log message used for every bug report.
const Message = "bug report"

// Field names. These are the contract with whatever consumes the logs, so
// changing any of them is a breaking change for downstream consumers.
const (
	// GroupKey is the slog group that holds every bug report field. Its
	// presence is what marks a log record as a bug report.
	GroupKey = "bug_report"

	FieldID          = "id"
	FieldTitle       = "title"
	FieldDescription = "description"
	FieldPriority    = "priority"
	FieldReporter    = "reporter"
	FieldComponent   = "component"
	FieldLabels      = "labels"
	FieldSteps       = "steps_to_reproduce"
	FieldExpected    = "expected"
	FieldActual      = "actual"
	FieldError       = "error"
	FieldMetadata    = "metadata"
	FieldSubject     = "subject"
)

// Priority is how urgently a bug should be addressed.
type Priority string

const (
	// NoPriority leaves prioritization to whoever triages the report.
	NoPriority Priority = "none"
	Urgent     Priority = "urgent"
	High       Priority = "high"
	Medium     Priority = "medium"
	Low        Priority = "low"
)

// Valid reports whether p is one of the defined priorities.
func (p Priority) Valid() bool {
	switch p {
	case NoPriority, Urgent, High, Medium, Low:
		return true
	default:
		return false
	}
}

var (
	// ErrMissingTitle is returned when a report has no title.
	ErrMissingTitle = errors.New("bug report title is required")

	// ErrInvalidPriority is returned when a report has an unrecognized priority.
	ErrInvalidPriority = errors.New("invalid bug report priority")
)

// Report is a single bug report. Only Title is required.
type Report struct {
	// ID uniquely identifies the report so consumers can deduplicate
	// redelivered log entries. Submit generates one if it is empty.
	ID string

	// Title is a one-line summary. Required.
	Title string

	// Description is free-form detail about the problem.
	Description string

	// Priority defaults to NoPriority when empty.
	Priority Priority

	// Reporter identifies who filed the report (an email, username, service name, etc).
	Reporter string

	// Subject identifies who or what the report concerns (org, project, etc).
	// Submit merges this over any subject carried in the context, so callers
	// usually only set what the context doesn't already know.
	Subject Subject

	// Component is the part of the system the bug concerns.
	Component string

	// Labels are free-form tags for categorization and routing.
	Labels []string

	// StepsToReproduce lists the steps that trigger the bug, in order.
	StepsToReproduce []string

	// Expected is what should have happened.
	Expected string

	// Actual is what happened instead.
	Actual string

	// Err is an error associated with the bug, if any.
	Err error

	// Metadata holds any additional context. Values must be JSON-serializable
	// to survive the trip through the logging pipeline.
	Metadata map[string]any
}

// Validate checks that the report is well-formed.
func (r *Report) Validate() error {
	if strings.TrimSpace(r.Title) == "" {
		return ErrMissingTitle
	}

	if r.Priority != "" && !r.Priority.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidPriority, r.Priority)
	}

	return nil
}

// Attr returns the report as a single "bug_report" group attribute. Empty
// fields are omitted. Submit uses this; it's exported for callers who want to
// emit the record through their own logger.
func (r *Report) Attr() slog.Attr {
	priority := r.Priority
	if priority == "" {
		priority = NoPriority
	}

	attrs := []any{
		slog.String(FieldID, r.ID),
		slog.String(FieldTitle, r.Title),
		slog.String(FieldPriority, string(priority)),
	}

	addString := func(key, val string) {
		if val != "" {
			attrs = append(attrs, slog.String(key, val))
		}
	}

	addStrings := func(key string, vals []string) {
		if len(vals) > 0 {
			attrs = append(attrs, slog.Any(key, vals))
		}
	}

	addString(FieldDescription, r.Description)
	addString(FieldReporter, r.Reporter)
	addString(FieldComponent, r.Component)
	addStrings(FieldLabels, r.Labels)
	addStrings(FieldSteps, r.StepsToReproduce)
	addString(FieldExpected, r.Expected)
	addString(FieldActual, r.Actual)

	if r.Err != nil {
		addString(FieldError, r.Err.Error())
	}

	if subject := r.Subject.attrs(); len(subject) > 0 {
		attrs = append(attrs, slog.Group(FieldSubject, subject...))
	}

	if len(r.Metadata) > 0 {
		meta := make([]any, 0, len(r.Metadata))
		for k, v := range r.Metadata {
			meta = append(meta, slog.Any(k, v))
		}

		attrs = append(attrs, slog.Group(FieldMetadata, meta...))
	}

	return slog.Group(GroupKey, attrs...)
}

// Submit validates the report and logs it, returning the report's ID.
//
// The record is logged at Info level so that user-filed reports don't trip
// error-rate alerting. It is logged even if ctx is muted, and is always marked
// sensitive so it is never routed to customer-facing log destinations; carry
// customer identifiers in the Subject instead.
func Submit(ctx context.Context, report Report) (string, error) {
	err := report.Validate()
	if err != nil {
		return "", err
	}

	report.Subject = SubjectFrom(ctx).Merge(report.Subject)

	if report.ID == "" {
		report.ID = uuid.NewString()
	}

	ctx = logger.WithSensitive(logger.WithMuted(ctx, false))

	logger.Get(ctx).LogAttrs(ctx, slog.LevelInfo, Message, report.Attr())

	return report.ID, nil
}
