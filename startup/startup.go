// Package startup provides utilities for application initialization and environment configuration.
//
// The primary use case is loading environment variables from files during application startup,
// which is useful for local development, container initialization, and testing scenarios.
package startup

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/xform"
)

// Option is a functional option for configuring environment loading behavior.
type Option func(*options)

// options holds the configuration for environment variable loading.
type options struct {
	// allowOverride determines whether environment variables loaded from files
	// can override existing environment variables. When false (default), existing
	// environment variables take precedence over file-based values.
	allowOverride bool
}

// WithAllowOverride configures whether loaded environment variables can override
// existing environment variables in the process.
//
// When allowOverride is true, variables from files will replace existing environment
// variables. When false (default), existing variables are preserved and file-based
// values are ignored for variables that already exist.
//
// Example:
//
//	// Allow files to override existing environment variables
//	err := ConfigureEnvironment(WithAllowOverride(true))
func WithAllowOverride(allowOverride bool) Option {
	return func(o *options) {
		o.allowOverride = allowOverride
	}
}

// ConfigureEnvironment loads environment variables from files specified in the ENV_FILE
// environment variable and sets them in the current process.
//
// The ENV_FILE variable should contain a semicolon-separated list of file paths.
// For example: ENV_FILE="/path/to/.env;/path/to/.env.local"
//
// By default, existing environment variables take precedence over file-based values.
// Use WithAllowOverride(true) to allow files to override existing variables.
//
// The function performs the following steps:
//  1. Parses the ENV_FILE environment variable (semicolon-separated file paths)
//  2. Loads all specified files using envutil.Loader
//  3. Sets environment variables from the loaded files (respecting override policy)
//
// If ENV_FILE is not set or empty, the function returns nil without error.
//
// Example usage:
//
//	// Load environment files, preserving existing variables
//	if err := ConfigureEnvironment(); err != nil {
//	    return fmt.Errorf("failed to configure environment: %w", err)
//	}
//
//	// Load environment files, allowing files to override existing variables
//	if err := ConfigureEnvironment(WithAllowOverride(true)); err != nil {
//	    return fmt.Errorf("failed to configure environment: %w", err)
//	}
func ConfigureEnvironment(opts ...Option) error {
	// Parse the ENV_FILE environment variable into a list of file paths.
	// The transformation pipeline: read ENV_FILE -> trim whitespace -> split on semicolon -> sanitize list
	envFiles := envutil.Map(
		envutil.Map(
			envutil.Map(
				envutil.String(context.Background(), "ENV_FILE"),
				xform.TrimString),
			xform.SplitString(";")),
		sanitizeEnvFileList).
		ValueOrElse(nil)

	return ConfigureEnvironmentFromFiles(envFiles, opts...)
}

// ConfigureEnvironmentFromFiles loads environment variables from the specified list of files
// and sets them in the current process.
//
// This function is similar to ConfigureEnvironment but accepts an explicit list of file paths
// instead of reading from the ENV_FILE environment variable. This is useful for testing or
// when you want programmatic control over which files to load.
//
// By default, existing environment variables take precedence over file-based values.
// Use WithAllowOverride(true) to allow files to override existing variables.
//
// The function performs the following steps:
//  1. Loads all specified files using envutil.Loader
//  2. Sets environment variables from the loaded files (respecting override policy)
//
// If the envFiles list is empty, the function returns nil without error.
//
// Example usage:
//
//	// Load specific environment file paths
//	filePaths := []string{"/path/to/.env", "/path/to/.env.local"}
//	if err := ConfigureEnvironmentFromFiles(filePaths); err != nil {
//	    return fmt.Errorf("failed to configure environment: %w", err)
//	}
//
//	// Load with override enabled
//	if err := ConfigureEnvironmentFromFiles(filePaths, WithAllowOverride(true)); err != nil {
//	    return fmt.Errorf("failed to configure environment: %w", err)
//	}
func ConfigureEnvironmentFromFiles(envFiles []string, opts ...Option) error {
	if len(envFiles) == 0 {
		// Nothing to do - no files specified
		return nil
	}

	cfg := getOptions(opts)

	// Create a new environment variable loader
	loader := envutil.NewLoader()

	// Load all specified files into the loader
	for _, file := range envFiles {
		_, err := loader.LoadFile(file)
		if err != nil {
			return fmt.Errorf("loading environment variables from file %q: %w", file, err)
		}
	}

	// Set environment variables from the loaded files, respecting the override policy
	for k, v := range loader.AsMap() {
		oldValue, exists := os.LookupEnv(k)
		if exists && (!cfg.allowOverride || oldValue == v) {
			// Skip if variable already exists and:
			// - override is not allowed, OR
			// - the value is the same as what we would set anyway
			continue
		}

		err := os.Setenv(k, v)
		if err != nil {
			return fmt.Errorf("setting environment variable %q=%q: %w", k, v, err)
		}
	}

	return nil
}

// sanitizeEnvFileList removes empty and whitespace-only entries from a list of file paths.
//
// This function is used to clean up the file path list after splitting ENV_FILE on semicolons,
// removing any empty strings that may result from trailing semicolons, double semicolons,
// or whitespace-only entries.
//
// Returns nil if the input is empty or if all entries are empty/whitespace after trimming.
func sanitizeEnvFileList(in []string) ([]string, error) {
	if len(in) == 0 {
		return nil, nil
	}

	out := make([]string, 0, len(in))

	for _, s := range in {
		s = strings.TrimSpace(s)

		if len(s) == 0 {
			continue
		}

		out = append(out, s)
	}

	if len(out) == 0 {
		return nil, nil
	}

	return out, nil
}

// getOptions applies the provided functional options and returns the resulting configuration.
//
// This helper function creates a default options struct and applies each provided Option
// function to it, building up the final configuration. Nil options are safely ignored.
func getOptions(opts []Option) *options {
	cfg := &options{}

	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	return cfg
}
