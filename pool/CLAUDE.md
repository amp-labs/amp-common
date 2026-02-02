# Package: pool

Generic object pooling with lifecycle management, idle cleanup, and Prometheus metrics.

## Usage

```go
import "github.com/amp-labs/amp-common/pool"

// Create pool
p := pool.New(func() (*DB, error) {
    return connectDB()
}, pool.WithName("db-pool"))

// Get and return objects
db, err := p.Get()
defer p.Put(db)

// Use object
result := db.Query(query)

// Close idle objects (e.g., every 5 minutes)
closed, err := p.CloseIdle(5 * time.Minute)

// Close entire pool
err := p.Close()
```

## Common Patterns

- Thread-safe pooling for any `io.Closer` objects
- Dynamic growth (creates objects on demand)
- `Get()` - Fetch from pool or create new
- `Put()` - Return to pool
- `CloseIdle()` - Remove unused objects
- Options: WithName (for metrics), WithCheckValid (health checks)
- Prometheus metrics: object counts, creation/close events, errors

## Gotchas

- Objects must implement `io.Closer`
- Pool periodically shuffles idle objects (prevents starvation)
- Timeouts: Get (5s), Put (10s), CloseIdle (30s)
- Validation function called before returning objects

## Related

- `closer` - Resource management utilities
