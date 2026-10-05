//nolint:err113 // Test file uses errors.New() for creating test errors
package bugreport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/amp-labs/amp-common/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLogs redirects the default slog logger to a JSON buffer for the
// duration of the test. Tests using it must not run in parallel.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer

	prev := slog.Default()

	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	t.Cleanup(func() { slog.SetDefault(prev) })

	return &buf
}

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	var rec map[string]any

	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))

	return rec
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		report Report
		err    error
	}{
		{"minimal", Report{Title: "broken"}, nil},
		{"missing title", Report{}, ErrMissingTitle},
		{"blank title", Report{Title: "  "}, ErrMissingTitle},
		{"valid priority", Report{Title: "broken", Priority: Urgent}, nil},
		{"invalid priority", Report{Title: "broken", Priority: "meh"}, ErrInvalidPriority},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.report.Validate()
			if tt.err == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.err)
			}
		})
	}
}

func TestSubmitFullReport(t *testing.T) { //nolint:paralleltest // mutates slog default
	buf := captureLogs(t)

	id, err := Submit(t.Context(), Report{
		ID:               "fixed-id",
		Title:            "Sync stalls",
		Description:      "Reads stop advancing",
		Priority:         High,
		Reporter:         "jane@example.com",
		Component:        "sync",
		Labels:           []string{"sync", "oauth"},
		StepsToReproduce: []string{"connect", "wait an hour"},
		Expected:         "reads continue",
		Actual:           "reads stop",
		Err:              errors.New("token expired"),
		Metadata:         map[string]any{"attempt": 3},
	})
	require.NoError(t, err)
	assert.Equal(t, "fixed-id", id)

	rec := decode(t, buf)
	assert.Equal(t, Message, rec["msg"])
	assert.Equal(t, "INFO", rec["level"])

	report, ok := rec[GroupKey].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, map[string]any{
		FieldID:          "fixed-id",
		FieldTitle:       "Sync stalls",
		FieldDescription: "Reads stop advancing",
		FieldPriority:    "high",
		FieldReporter:    "jane@example.com",
		FieldComponent:   "sync",
		FieldLabels:      []any{"sync", "oauth"},
		FieldSteps:       []any{"connect", "wait an hour"},
		FieldExpected:    "reads continue",
		FieldActual:      "reads stop",
		FieldError:       "token expired",
		FieldMetadata:    map[string]any{"attempt": float64(3)},
	}, report)
}

func TestSubmitMinimalReport(t *testing.T) { //nolint:paralleltest // mutates slog default
	buf := captureLogs(t)

	id, err := Submit(t.Context(), Report{Title: "broken"})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	report, ok := decode(t, buf)[GroupKey].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, map[string]any{
		FieldID:       id,
		FieldTitle:    "broken",
		FieldPriority: "none",
	}, report)
}

func TestSubmitInvalidReportLogsNothing(t *testing.T) { //nolint:paralleltest // mutates slog default
	buf := captureLogs(t)

	_, err := Submit(t.Context(), Report{})
	require.ErrorIs(t, err, ErrMissingTitle)
	assert.Empty(t, buf.String())
}

func TestSubmitIgnoresMute(t *testing.T) { //nolint:paralleltest // mutates slog default
	buf := captureLogs(t)

	_, err := Submit(logger.WithMuted(t.Context(), true), Report{Title: "broken"})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestSubmitNeverRoutesToCustomers(t *testing.T) { //nolint:paralleltest // mutates slog default
	buf := captureLogs(t)

	ctx := logger.WithCustomerId(context.Background(), "cust-1")
	ctx = logger.WithRoutingToBuilder(ctx, "proj-1")

	_, err := Submit(ctx, Report{Title: "broken"})
	require.NoError(t, err)

	rec := decode(t, buf)
	assert.NotContains(t, rec, "customer_id")
	assert.NotContains(t, rec, "log_project")
}

// Consumer-defined subject keys, as a program using this package would declare them.
const (
	testOrg     Key = "org_id"
	testProject Key = "project_id"
)

func TestSubmitMergesContextSubject(t *testing.T) { //nolint:paralleltest // mutates slog default
	buf := captureLogs(t)

	ctx := WithAttribute(t.Context(), testOrg, "org-1")
	ctx = WithSubject(ctx, Subject{testProject: "proj-ctx"})

	_, err := Submit(ctx, Report{
		Title:   "broken",
		Subject: Subject{testProject: "proj-report", "empty": ""},
	})
	require.NoError(t, err)

	report, ok := decode(t, buf)[GroupKey].(map[string]any)
	require.True(t, ok)

	// Report values win over context values, and empty values are dropped.
	assert.Equal(t, map[string]any{
		"org_id":     "org-1",
		"project_id": "proj-report",
	}, report[FieldSubject])
}

func TestSubmitOmitsEmptySubject(t *testing.T) { //nolint:paralleltest // mutates slog default
	buf := captureLogs(t)

	_, err := Submit(t.Context(), Report{Title: "broken", Subject: Subject{testOrg: ""}})
	require.NoError(t, err)

	report, ok := decode(t, buf)[GroupKey].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, report, FieldSubject)
}

func TestWithSubjectDoesNotMutateParent(t *testing.T) {
	t.Parallel()

	parent := WithAttribute(t.Context(), testOrg, "org-1")
	child := WithAttribute(parent, testProject, "proj-1")

	assert.Equal(t, Subject{testOrg: "org-1"}, SubjectFrom(parent))
	assert.Equal(t, Subject{testOrg: "org-1", testProject: "proj-1"}, SubjectFrom(child))
}
