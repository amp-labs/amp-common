# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in amp-common, please report it responsibly.

### Contact

**Do NOT create a public GitHub issue for security vulnerabilities.**

Instead, please report security issues to:

- **Email**: <security@withampersand.com>

### What to Include

When reporting a vulnerability, please include:

- Description of the vulnerability
- Steps to reproduce the issue
- Affected versions
- Potential impact
- Any suggested fixes (if you have them)

## Supported Versions

We actively support and maintain security updates for:

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |

**Go Version:** This project requires Go 1.25.5 or later for security and stability.

## Security Best Practices

When working with amp-common, follow these security guidelines:

### 1. Never Commit Secrets

**Never commit sensitive data to version control:**

- API keys
- Passwords
- Private keys
- Connection strings
- OAuth tokens

**Bad:**

```go
const APIKey = "sk_live_abc123..." // Never hardcode secrets
```

**Good:**

```go
apiKey := envutil.String("API_KEY", envutil.Required()).Value()
```

Add sensitive files to `.gitignore`:

```gitignore
.env
.env.local
secrets/
*.key
*.pem
credentials.json
```

### 2. Be Careful With Recording When Reading Secrets

Recording is **disabled by default** in envutil, so secrets are not captured unless you explicitly enable recording. If you do enable recording for testing or debugging, be careful when reading secrets:

```go
// Recording is disabled by default - secrets are safe
apiKey := envutil.String(ctx, "API_KEY").Value()

// If you've enabled recording for tests/debugging:
// envutil.EnableRecording(true)
//
// And need to read secrets, temporarily disable it:
envutil.EnableRecording(false)
apiKey := envutil.String(ctx, "API_KEY").Value()
envutil.EnableRecording(true)  // Re-enable if needed
```

**Key point:** Recording is off by default. Only worry about this if you're explicitly using `EnableRecording(true)` in your code.

### 3. Disable Recording When Testing with Secrets

The `envutil` package supports recording environment variable reads for testing. **Disable recording when working with secrets:**

```go
// In test setup
envutil.WithRecording(false)()  // Disable recording

// Now safe to read secrets
apiKey := envutil.Secret("API_KEY").Value()

// Recording is disabled, so secrets won't be captured
```

### 4. Separate Configuration from Secrets

Use separate `.env` files for configuration and secrets:

```bash
# .env - Safe to commit (configuration)
PORT=8080
LOG_LEVEL=info
ENVIRONMENT=production

# .env.secrets - NEVER commit (secrets)
DATABASE_PASSWORD=...
API_KEY=...
JWT_SECRET=...
```

Load both files:

```go
startup.ConfigureEnvironmentFromFiles(".env", ".env.secrets")
```

### 5. Private Dependencies

This repository uses private GitHub repositories. Ensure proper authentication:

**SSH Authentication (Recommended):**

```bash
# Configure Git to use SSH
git config --global url."git@github.com:".insteadOf "https://github.com/"

# Set GOPRIVATE
export GOPRIVATE=github.com/amp-labs/*
```

**Never:**

- Commit GitHub personal access tokens
- Use HTTPS with embedded credentials
- Share SSH private keys

### 6. Context Overrides Can Leak Secrets

Be cautious when using context overrides in `envutil`:

```go
// Context overrides bypass Secret() protection
ctx := envutil.WithOverride(context.Background(), "API_KEY", "secret-value")

// This reads from context, not environment
apiKey := envutil.Secret("API_KEY").ValueContext(ctx)
// Value is NOT redacted because it came from context override
```

**Recommendation:** Avoid context overrides for secrets. Use them only for non-sensitive configuration in tests.

### 7. Validate User Input

Always validate and sanitize user input:

```go
import "github.com/amp-labs/amp-common/sanitize"

// Sanitize user input before use
userInput := sanitize.AlphaNumeric(rawInput)

// Validate before processing
if err := validate.Do(ctx, userStruct); err != nil {
    return fmt.Errorf("validation failed: %w", err)
}
```

### 8. Use Secure Defaults

- Enable TLS for network connections
- Use secure random number generation (`crypto/rand`, not `math/rand`)
- Set appropriate timeouts for HTTP clients
- Use context with deadlines for external operations

### 9. Secret Management in Production

For production deployments, consider using dedicated secret management:

- **Kubernetes Secrets** - For Kubernetes deployments
- **AWS Secrets Manager** - For AWS infrastructure
- **HashiCorp Vault** - For multi-cloud environments
- **Google Secret Manager** - For GCP infrastructure

Load secrets at runtime, not build time:

```go
// Good: Runtime secret loading
secret := secretsManager.GetSecret("api-key")

// Bad: Build-time embedding
const apiKey = "..." // Never do this
```

### 10. Dependency Security

- Keep dependencies up to date
- Review Dependabot security alerts promptly
- Run `go mod tidy` regularly
- Use `go list -m -versions` to check for updates

## Vulnerability Disclosure Timeline

When a vulnerability is reported:

1. **24 hours**: Initial response acknowledging the report
2. **72 hours**: Preliminary assessment and severity classification
3. **7 days**: Fix development and testing
4. **14 days**: Release of security patch
5. **30 days**: Public disclosure (coordinated with reporter)

Timeline may vary based on severity and complexity.

## Security Update Process

When we release security updates:

1. **Critical vulnerabilities**: Immediate patch release
2. **High severity**: Patch within 7 days
3. **Medium severity**: Patch within 14 days
4. **Low severity**: Include in next regular release

Security patches are:

- Released as patch versions (e.g., v1.2.3 → v1.2.4)
- Documented in release notes
- Communicated to users via GitHub security advisories

## Keeping amp-common Secure

We take security seriously. Our practices include:

- Regular security audits
- Dependency vulnerability scanning
- Automated security testing
- Code review requirements for all changes
- Minimal third-party dependencies
- Clear separation of concerns

## Additional Resources

- [Go Security Best Practices](https://go.dev/security/best-practices)
- [OWASP Go Secure Coding Practices](https://owasp.org/www-project-go-secure-coding-practices-guide/)
- [envutil Security Documentation](envutil/README.md#security)

Thank you for helping keep amp-common secure!
