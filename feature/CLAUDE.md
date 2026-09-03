# Package: feature

Feature flags: switches defined in code whose rollout changes at runtime
without a restart. Percentage rollouts are deterministic: the bucket is
xxh3(flag name + salt + dimension + subject value) mod 10000. The library is
domain-neutral: only the `Host`, `Stage`, and `Region` dimensions are built
in, and consumers declare the rest. `README.md` in this directory is the full guide.

## Usage

```go
import "github.com/amp-labs/amp-common/feature"

// Declare the dimensions your program has. The library ships only Host
// (hostname), Stage (stage.Current), and Region (region.Current).
const (
    User   feature.Dimension = "user"
    Tenant feature.Dimension = "tenant"
)

// Define once, in code. The default applies until something overrides it.
var NewCheckout = feature.Define("new-checkout", feature.Off(),
    feature.Description("Use the rewritten checkout flow"))

// Middleware: attach the subject once per request.
ctx = feature.WithAttribute(ctx, User, userID)
ctx = feature.WithAttribute(ctx, Tenant, tenantID)

// Anywhere below.
if NewCheckout.Enabled(ctx) {
    // ...
}

// Change at runtime from code...
err := NewCheckout.Set(feature.Off().
    Allow(Tenant, "t_beta_1").
    Percent(User, 25))

// ...or from the environment, a file, and HTTP.
// Env:  FEATURE_NEW_CHECKOUT=on | off | 25% user | {json}
// File: {"new-checkout": "25% user", ...}, polled for changes
go feature.Watch(ctx, feature.Multi(
    feature.EnvSource(""),
    feature.FileSource("/etc/flags/flags.json"),
))
mux.Handle("/internal/flags/",
    http.StripPrefix("/internal/flags", feature.Handler(nil)))
```

## Common Patterns

- Builders: `Off()`, `On()`, `.Allow(dim, values...)`,
  `.Deny(dim, values...)`, `.Percent(dim, pct)`, `.PercentWithSalt`,
  `.WithDefault`
- Evaluation order: Denied > Allowed > any Percentage (OR) > Default
- Built-in dimensions: `Host` (hostname), `Stage` (`stage.Current`),
  `Region` (`region.Current`); programs declare the rest as `Dimension` consts
- Subject precedence: explicit `EnabledFor` subject > context subject >
  `SetGlobalAttribute` > automatic values
- `Flag.Evaluate` returns a `Result` with `Reason` and `Bucket` for "why"
  questions; `Bucket()` reproduces the bucketing elsewhere
- Sources: `FileSource` (JSON, polled), `EnvSource`, `Static`, `Multi` (later
  sources win); all call `Registry.Load`
- Handler routes: `GET /`, `GET /{name}`, `PUT /{name}`, `DELETE /{name}`,
  `POST /{name}/evaluate`
- Tests: `flag.Set(feature.On())` plus `t.Cleanup(flag.Reset)`, or
  `NewRegistry()` for full isolation
- Metrics: `feature_flag_evaluations_total{flag,enabled,reason}`,
  `feature_flag_changes_total{flag,source}`,
  `feature_flag_undefined_lookups_total`

## Gotchas

- Keep the library domain-neutral: do not add dimensions such as users or
  tenants here; consumers own those
- Rollouts are data, not closures: fixed rule kinds, JSON-serializable, and
  logged on every change
- A subject missing a rule's dimension never matches that rule, and an empty
  string counts as missing
- Undefined flag names evaluate to false, are counted, and are logged once
- `Registry.Load` (what Sources call) is authoritative: flags absent from the
  set revert to defaults and manual `Set` calls are clobbered. If you run a
  Source, point the admin path at its backing store
- `Host` defaults to the hostname (pod name), so a host percentage selects a
  different set of pods after each deploy
- Renaming a flag or changing a salt reshuffles buckets; raising a percentage
  is monotonic
- `Handler` does no authentication; mount it on an internal listener
- `Define` panics on duplicate or empty names and on invalid defaults, since
  those are startup programming errors

## Related

- `stage` - supplies the automatic `Stage` dimension
- `region` - supplies the automatic `Region` dimension
- `hashing` - shares the xxh3 dependency
