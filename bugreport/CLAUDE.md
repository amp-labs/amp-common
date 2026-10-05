# Package: bugreport

Emits ticket-like bug reports as standardized slog records, for a log-based
pipeline (log sink -> cloud function -> issue tracker) to pick up.

## Usage

```go
import "github.com/amp-labs/amp-common/bugreport"

// Consumers define their own subject keys.
const (
    Org         bugreport.Key = "org_id"
    Integration bugreport.Key = "integration_id"
)

// Middleware attaches identifiers as they become known...
ctx = bugreport.WithAttribute(ctx, Org, orgID)

// ...and every report submitted downstream includes them.
id, err := bugreport.Submit(ctx, bugreport.Report{
    Title:       "Sync stalls after token refresh", // required
    Description: "Reads stop advancing once the token is refreshed.",
    Priority:    bugreport.High,
    Reporter:    "jane@example.com",
    Labels:      []string{"sync"},
    Subject:     bugreport.Subject{Integration: integrationID},
})
```

## Log shape

Message `"bug report"`, level Info, with every field under a `bug_report`
group: `id`, `title`, `priority` always; `description`, `reporter`,
`subject`, `component`, `labels`, `steps_to_reproduce`, `expected`, `actual`, `error`,
`metadata` when set. Filter in GCP with `jsonPayload.bug_report.id:*`.

## Gotchas

- Field names (the `Field*` constants) are a contract with downstream consumers
- `id` is a UUID generated per report so consumers can dedupe redelivered logs
- Submit ignores a muted context and always marks the record sensitive, so
  `customer_id`/`log_project` are dropped and it never routes to customers;
  carry customer identifiers in the `Subject`
- Subject keys are consumer-defined (`bugreport.Key`); the package ships none.
  The report's own `Subject` is merged over the context's; empty values are dropped
- Logged at Info so user-filed reports don't trip error-rate alerts
