// Package feature provides feature flags: named switches defined in code whose
// rollout can be changed at runtime without restarting the process.
//
// A flag has three parts with different lifetimes:
//
//   - The definition lives in code and never changes at runtime. It carries the
//     flag's name and a default Rollout, which is what the flag does when nothing
//     has overridden it (including when a Source is unreachable).
//   - The Rollout is pure data that changes at runtime and says who gets the flag.
//     It can be set programmatically, loaded from a file or the environment, or
//     changed through the HTTP Handler.
//   - The Subject is who is asking, expressed as values along named Dimensions.
//     Programs define the dimensions that exist in their domain, such as users
//     or tenants; only Host, Stage, and Region are built in.
//
// Percentage rollouts are deterministic. A subject's bucket is derived from a
// hash of the flag name, the rule's salt, the dimension, and the subject's value
// for that dimension. The same subject therefore gets the same answer on every
// process with no coordination, raising a percentage never removes anyone who
// already had the flag, and being inside the rollout for one flag says nothing
// about any other flag.
//
// Typical usage:
//
//	// Dimensions are just names. Declare the ones your program has.
//	const (
//		User   feature.Dimension = "user"
//		Tenant feature.Dimension = "tenant"
//	)
//
//	var NewCheckout = feature.Define("new-checkout", feature.Off(),
//		feature.Description("Use the rewritten checkout flow"))
//
//	// In middleware, once per request:
//	ctx = feature.WithAttribute(ctx, User, userID)
//
//	// Anywhere below:
//	if NewCheckout.Enabled(ctx) {
//		// ...
//	}
//
//	// At runtime, from code or via PUT on the Handler:
//	err := NewCheckout.Set(feature.Off().
//		Allow(Tenant, "t_beta_1").
//		Percent(User, 25))
//
// See README.md in this directory for a full guide.
package feature
