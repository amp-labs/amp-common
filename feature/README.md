# feature

Feature flags for Go services: switches defined in code whose rollout can be
changed at runtime without a restart, including deterministic percentage
rollouts to any population you can name (hosts, users, tenants, ...).

```go
var NewCheckout = feature.Define("new-checkout", feature.Off())

if NewCheckout.Enabled(ctx) {
    // new path
}
```

- [Concepts](#concepts)
- [Quick start](#quick-start)
- [Defining flags](#defining-flags)
- [Subjects and dimensions](#subjects-and-dimensions)
- [Rollouts](#rollouts)
- [How percentage rollouts work](#how-percentage-rollouts-work)
- [Changing flags at runtime](#changing-flags-at-runtime)
- [Observability](#observability)
- [Testing](#testing)
- [FAQ](#faq)

## Concepts

A flag has three parts with different lifetimes.

| Part       | Lives in | Changes          | Says                  |
| ---------- | -------- | ---------------- | --------------------- |
| Definition | code     | never at runtime | name, default rollout |
| Rollout    | data     | at runtime       | who gets the flag     |
| Subject    | request  | per call         | who is asking         |

**Definition.** `feature.Define` registers a flag with a *default rollout*.
The default is what the flag does when nothing has overridden it, including
when a configuration source is unreachable.

**Rollout.** A `Rollout` is a plain struct that says who gets the flag: a
denylist, an allowlist, percentages, and a fallback. Because it is data rather
than code, it can come from a file, an environment variable, or an HTTP
request, and every change can be logged and diffed.

**Subject.** A `Subject` is who is asking, expressed as values along named
*dimensions*: `{"host": "api-7f9c", "user": "u_123"}`. A dimension is any
string. The library defines only the three that every deployment has: `Host`,
`Stage`, and `Region`. You define the rest.

## Quick start

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/amp-labs/amp-common/feature"
)

// 1. Name the dimensions that exist in your domain.
const (
    User   feature.Dimension = "user"
    Tenant feature.Dimension = "tenant"
)

// 2. Define flags at package level with their code defaults.
var (
    NewCheckout = feature.Define("new-checkout", feature.Off(),
        feature.Description("Use the rewritten checkout flow"),
        feature.Owner("payments"))

    FastSync = feature.Define("fast-sync", feature.Off().
        Allow(feature.Stage, "dev", "staging").
        Percent(Tenant, 5))
)

func main() {
    ctx := context.Background()

    // 3. Feed the registry from the environment and a file, and expose it.
    go func() {
        err := feature.Watch(ctx, feature.Multi(
            feature.EnvSource(""),
            feature.FileSource("/etc/flags/flags.json"),
        ))
        if err != nil {
            log.Fatal(err)
        }
    }()

    mux := http.NewServeMux()
    mux.Handle("/internal/flags/",
        http.StripPrefix("/internal/flags", feature.Handler(nil)))
    // ...
}

// 4. Attach the subject once per request, in middleware.
func withSubject(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := feature.WithAttribute(r.Context(), User, userIDFrom(r))
        ctx = feature.WithAttribute(ctx, Tenant, tenantIDFrom(r))
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// 5. Check flags anywhere below. No IDs need to be threaded through.
func checkout(ctx context.Context) {
    if NewCheckout.Enabled(ctx) {
        // ...
    }
}
```

## Defining flags

```go
var NewCheckout = feature.Define("new-checkout", feature.Off(),
    feature.Description("Use the rewritten checkout flow"),
    feature.Owner("payments"))
```

- The name is the flag's stable identity. It is also part of the hash input
  for percentage rollouts, so renaming a flag reshuffles who is inside.
- The second argument is the code default. `feature.Off()` is the usual
  choice for a new flag.
- `Define` panics if the name is empty or already defined, or if the default
  is invalid. All three are programming errors, and panicking at startup is
  the same trade-off `prometheus.MustRegister` makes.
- `Description` and `Owner` are optional and only appear in listings.

Flags live in a `Registry`. `feature.Define` uses the package-level default
registry, which is what most programs want. `feature.NewRegistry()` creates
an isolated one, useful in tests or when embedding several independent flag
sets in one process.

## Subjects and dimensions

A `Dimension` is a string naming a population you might slice: users,
tenants, hosts. Declare the ones your program has:

```go
const (
    User   feature.Dimension = "user"
    Tenant feature.Dimension = "tenant"
)
```

Three dimensions are built in because every deployment has them:

| Dimension        | Value when not set explicitly                     |
| ---------------- | ------------------------------------------------- |
| `feature.Host`   | the hostname, which is the pod name on Kubernetes |
| `feature.Stage`  | `stage.Current(ctx)` from the `stage` package     |
| `feature.Region` | `region.Current(ctx)` from the `region` package   |

A percentage of `Host` is a percentage of your fleet. See the note under
[How percentage rollouts work](#how-percentage-rollouts-work) about what
happens on redeploy.

### Attaching a subject

The subject usually rides in the context. Set it once, in middleware, and
code deeper in the stack calls `flag.Enabled(ctx)` without knowing what a
user or tenant is:

```go
ctx = feature.WithAttribute(ctx, User, "u_123")
ctx = feature.WithSubject(ctx, feature.Subject{Tenant: "t_9", User: "u_123"})
```

For one-off checks, pass a subject explicitly:

```go
flag.EnabledFor(ctx, feature.Subject{Tenant: "t_9"})
```

Process-wide attributes, such as a custom host identity, go on the registry:

```go
feature.Default().SetGlobalAttribute(feature.Host, os.Getenv("POD_NAME"))
```

When the same dimension is set in more than one place, the most specific
wins: explicit subject, then context, then global attributes, then the
automatic `Host`, `Stage`, and `Region` values.

An empty string counts as "not set". A rule about a dimension the subject
does not have never matches, so an anonymous request cannot accidentally
collapse into a single bucket.

## Rollouts

A `Rollout` is data:

```go
type Rollout struct {
    Denied      []Target     // always off for these
    Allowed     []Target     // always on for these
    Percentages []Percentage // on for a deterministic fraction
    Default     bool         // everyone else
}
```

Evaluation runs in that order and the first decisive rule wins:

1. If the subject matches any **Denied** target, the flag is off.
2. If it matches any **Allowed** target, the flag is on.
3. If it falls inside **any** percentage, the flag is on.
4. Otherwise the result is **Default**.

The zero value is off for everyone. A rule about a dimension the subject
lacks is skipped.

### Building rollouts in code

Builders return copies, so rollouts compose without aliasing:

```go
feature.Off()                                  // off for everyone
feature.On()                                   // on for everyone
feature.Off().Percent(User, 25)                // 25% of users
feature.Off().Percent(feature.Host, 10)        // 10% of the fleet
feature.On().Deny(Tenant, "t_bad")             // everyone except one tenant
feature.Off().Allow(feature.Stage, "dev")      // dev only
feature.Off().PercentWithSalt(User, 25, "v2")  // same 25%, reshuffled

feature.Off().
    Allow(Tenant, "t_beta_1", "t_beta_2"). // beta tenants always in
    Deny(User, "u_problem").               // one user always out
    Percent(User, 25)                      // and 25% of everyone else
```

Multiple percentages are OR'd: a subject inside any of them gets the flag.
"10% of hosts **and** 25% of users" is deliberately not expressible in one
rollout; use two flags.

### Rollouts as JSON

The same rollout as JSON, which is what files, the environment, and the HTTP
handler accept:

```json
{
  "denied":  [{"dimension": "user", "values": ["u_problem"]}],
  "allowed": [{"dimension": "tenant", "values": ["t_beta_1", "t_beta_2"]}],
  "percentages": [{"dimension": "user", "percent": 25}],
  "default": false
}
```

Anywhere a rollout is expected as a JSON string, a shorthand is accepted:

| Shorthand                       | Meaning                            |
| ------------------------------- | ---------------------------------- |
| `on`, `true`, `enabled`, `1`    | on for everyone                    |
| `off`, `false`, `disabled`, `0` | off for everyone                   |
| `25% user`, `25% of user`       | 25 percent of the `user` dimension |

`feature.ParseRollout` implements it; `Rollout.Validate` checks ranges.

## How percentage rollouts work

For a rule `Percent(dim, P)` and a subject whose value for `dim` is `v`:

```text
bucket  = xxh3(flagName + salt + dim + v) mod 10000
enabled = bucket < P * 100
```

That one line gives the properties you want from a rollout:

- **Deterministic.** The same subject gets the same answer on every host,
  with no coordination and no storage.
- **Independent per flag.** The flag name is in the hash, so being in the
  10% for one flag says nothing about another.
- **Monotonic.** Raising 10% to 25% keeps everyone who was in. Lowering only
  removes from the top.
- **Reshuffleable.** Change the salt to pick a fresh cohort without renaming
  the flag.
- **Fine-grained.** 10,000 buckets give 0.01% resolution.

`feature.Bucket(nil, name, salt, dim, value)` computes the same bucket, so
you can reproduce the assignment in SQL, another language, or a support tool.

**Host percentages and redeploys.** The default `Host` value is the hostname,
which on Kubernetes changes every time a pod is replaced. A 10% host rollout
is therefore a different 10% after each deploy. If you need stickiness, set
`Host` to something stable such as a StatefulSet ordinal or a node label.

## Changing flags at runtime

### In process

```go
err := NewCheckout.Set(feature.Off().Percent(User, 25))
NewCheckout.Reset() // back to the code default
```

`Set` validates and returns an error wrapping `feature.ErrInvalidRollout`
for out-of-range percentages or rules without a dimension.

### From a source

A `Source` pushes complete sets of overrides into the registry whenever they
change. Run one with `Watch`, which blocks until the context is done:

```go
go func() {
    err := feature.Watch(ctx, feature.FileSource("/etc/flags/flags.json"))
    // err is nil on shutdown; anything else is a real failure
}()
```

A source is **authoritative**: every time it delivers a set, flags absent
from that set revert to their code defaults, and anything installed with
`Set` is replaced. If you run a source, make your admin tooling write to the
source's backing store rather than calling `Set`.

`FileSource(path, ...)` watches a JSON file mapping flag names to rollouts,
in object or shorthand form:

```json
{
  "new-checkout": "25% user",
  "fast-sync": {
    "allowed": [{"dimension": "tenant", "values": ["t_beta_1"]}],
    "percentages": [{"dimension": "tenant", "percent": 5}]
  },
  "legacy-export": "off"
}
```

It re-reads the file when its contents change, checking every ten seconds by
default (`feature.PollInterval` adjusts this). A missing file means no
overrides. A file that fails to parse is logged and ignored, and the
previous state stays in effect. Entries with invalid rollouts are logged and
skipped without touching the other flags.

`EnvSource(prefix)` reads the environment once at startup. With the default
prefix `FEATURE_`, the variable name is the upper-cased flag name with dashes
turned into underscores:

```sh
FEATURE_NEW_CHECKOUT=on
FEATURE_FAST_SYNC="5% tenant"
FEATURE_LEGACY_EXPORT='{"default": false}'
```

`Multi(sources...)` merges several sources, later ones winning on conflicts,
so environment plus file is a natural pairing. `Static(map)` is a fixed set,
handy in tests. Anything implementing `Source` plugs in the same way, so a
database or remote configuration service is a small adapter:

```go
type Source interface {
    // Run blocks until ctx is done, calling apply with the complete set of
    // overrides each time it changes. Return nil on shutdown.
    Run(ctx context.Context, apply feature.ApplyFunc) error
}
```

### Over HTTP

`feature.Handler` exposes a registry for inspection and changes. Routes are
relative to its root, so mount it with `http.StripPrefix`:

```go
mux.Handle("/internal/flags/",
    http.StripPrefix("/internal/flags", feature.Handler(nil)))
```

| Method and path         | Effect                                      |
| ----------------------- | ------------------------------------------- |
| `GET /`                 | list every flag: default, current, override |
| `GET /{name}`           | one flag                                    |
| `PUT /{name}`           | set the rollout from the JSON body          |
| `DELETE /{name}`        | reset to the code default                   |
| `POST /{name}/evaluate` | evaluate for the subject in the body        |

```sh
curl -X PUT localhost:8080/internal/flags/new-checkout -d '"25% user"'
curl -X PUT localhost:8080/internal/flags/new-checkout \
  -d '{"allowed":[{"dimension":"tenant","values":["t_beta_1"]}]}'
curl -X DELETE localhost:8080/internal/flags/new-checkout
curl -X POST localhost:8080/internal/flags/new-checkout/evaluate \
  -d '{"user":"u_123"}'
```

The handler does no authentication. Mount it on an internal listener or
behind your own middleware. Changes made this way affect only the process
that received the request, and a running source replaces them on its next
update.

## Observability

### Why did this subject get that answer?

`Evaluate` returns the full story:

```go
result := NewCheckout.Evaluate(ctx, feature.Subject{User: "u_123"})
// result.Enabled   false
// result.Reason    "default"
// result.Dimension ""
// result.Bucket    7312   (of 10000; a 25% rule covers 0 to 2499)
```

| Reason       | Meaning                                                  |
| ------------ | -------------------------------------------------------- |
| `denied`     | matched a Denied target                                  |
| `allowed`    | matched an Allowed target                                |
| `percentage` | fell inside a percentage; `Bucket` says where            |
| `default`    | no rule matched; `Bucket` is from the last percentage    |
| `undefined`  | the flag was never defined                               |

The same result is available over HTTP via `POST /{name}/evaluate`.

### Metrics

| Metric                                 | Labels                       |
| -------------------------------------- | ---------------------------- |
| `feature_flag_evaluations_total`       | `flag`, `enabled`, `reason`  |
| `feature_flag_changes_total`           | `flag`, `source`             |
| `feature_flag_undefined_lookups_total` | none                         |

`source` is one of `set`, `reset`, or `load`. Rollout progress per flag is
`sum(rate(feature_flag_evaluations_total[5m])) by (flag, enabled)`.

### Logs

Every rollout change is logged at Info with the flag, the source of the
change, and the previous and new rollout as JSON. Evaluating a flag that was
never defined logs a warning once per name.

## Testing

Flip a flag for one test and restore it afterwards:

```go
func TestNewCheckout(t *testing.T) {
    require.NoError(t, NewCheckout.Set(feature.On()))
    t.Cleanup(NewCheckout.Reset)
    // ...
}
```

Flags on the default registry are process-global, so tests that flip them
should not run in parallel with tests that read them. For full isolation,
give the code under test its own registry:

```go
reg := feature.NewRegistry()
flag := reg.Define("new-checkout", feature.Off())
```

`feature.NewRegistry(feature.WithHasher(...))` swaps the hash function when
a test needs to force subjects into or out of a percentage.

## FAQ

**Why are rollouts data instead of functions?** So they can be changed from
outside the process: loaded from a file, set over HTTP, stored in a
database, logged, and diffed. A `func(ctx) bool` can do none of that. The
cost is a fixed set of rule kinds, and the four here cover the real cases.

**How do I express "10% of hosts AND 25% of users"?** With two flags,
checked together. Percentages within one rollout are OR'd on purpose so
that the common cases stay readable.

**What happens when the flag file or remote store is unavailable?** Every
flag evaluates its code default. Nothing fails.

**What if I check a flag that was never defined?** `feature.Enabled(ctx,
name)` returns false, increments `feature_flag_undefined_lookups_total`, and
logs once. Prefer holding the `*Flag` returned by `Define`, which cannot be
undefined.

**Can a flag return something other than a bool?** Not in this package.
Multivariate flags, scheduled rollouts, and sticky assignment when a
subject's identity changes all layer on top of this data model without
changing it, but none are implemented.
