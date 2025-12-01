package telemetry

import (
	"os"
	"testing"
)

func TestLoadConfigFromEnv_GKEDetection(t *testing.T) {
	// Store original environment
	originalHost := os.Getenv("KUBERNETES_SERVICE_HOST")
	originalEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")

	// Clean up after test
	defer func() {
		if originalHost == "" {
			_ = os.Unsetenv("KUBERNETES_SERVICE_HOST")
		} else {
			t.Setenv("KUBERNETES_SERVICE_HOST", originalHost)
		}

		if originalEndpoint == "" {
			_ = os.Unsetenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
		} else {
			t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", originalEndpoint)
		}
	}()

	tests := []struct {
		name             string
		kubernetesHost   string
		expectedEndpoint string
		customEndpoint   string
	}{
		{
			name:             "GKE environment detected",
			kubernetesHost:   "10.0.0.1",
			expectedEndpoint: "http://opentelemetry-collector.opentelemetry.svc.cluster.local:4318",
		},
		{
			name:             "Non-GKE environment",
			kubernetesHost:   "",
			expectedEndpoint: "",
		},
		{
			name:             "Custom endpoint overrides GKE default",
			kubernetesHost:   "10.0.0.1",
			customEndpoint:   "http://custom-collector:4318",
			expectedEndpoint: "http://custom-collector:4318",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set up environment
			_ = os.Unsetenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")

			if test.kubernetesHost != "" {
				t.Setenv("KUBERNETES_SERVICE_HOST", test.kubernetesHost)
			} else {
				_ = os.Unsetenv("KUBERNETES_SERVICE_HOST")
			}

			if test.customEndpoint != "" {
				t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", test.customEndpoint)
			}

			// Load config
			config, err := LoadConfigFromEnv(t.Context(), "dev")
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Verify endpoint
			if config.Endpoint != test.expectedEndpoint {
				t.Errorf("Expected endpoint %s, got %s", test.expectedEndpoint, config.Endpoint)
			}
		})
	}
}

func TestLoadConfigFromEnv_DefaultValues(t *testing.T) { //nolint:paralleltest
	// Store and clean original environment
	originalEnabled := os.Getenv("OTEL_ENABLED")
	originalServiceName := os.Getenv("OTEL_SERVICE_NAME")
	originalServiceVersion := os.Getenv("OTEL_SERVICE_VERSION")
	originalEnvironment := os.Getenv("ENVIRONMENT")

	defer func() {
		restoreEnv("OTEL_ENABLED", originalEnabled)
		restoreEnv("OTEL_SERVICE_NAME", originalServiceName)
		restoreEnv("OTEL_SERVICE_VERSION", originalServiceVersion)
		restoreEnv("ENVIRONMENT", originalEnvironment)
	}()

	// Clear environment
	_ = os.Unsetenv("OTEL_ENABLED")
	_ = os.Unsetenv("OTEL_SERVICE_NAME")
	_ = os.Unsetenv("OTEL_SERVICE_VERSION")
	_ = os.Unsetenv("ENVIRONMENT")

	config, err := LoadConfigFromEnv(t.Context(), "test")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test defaults
	if config.Enabled != false {
		t.Errorf("Expected Enabled to be false, got %t", config.Enabled)
	}

	if config.ServiceVersion != defaultServiceVersion {
		t.Errorf("Expected ServiceVersion %s, got %s", defaultServiceVersion, config.ServiceVersion)
	}

	if config.Environment != "test" {
		t.Errorf("Expected Environment 'test', got %s", config.Environment)
	}
}

func restoreEnv(key, value string) {
	if value == "" {
		_ = os.Unsetenv(key)
	} else {
		_ = os.Setenv(key, value)
	}
}
