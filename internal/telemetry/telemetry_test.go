package telemetry

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeOTLPEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hostport string
		urlPath  string
		insecure bool
		resolved string
		wantErr  bool
	}{
		{"default localhost", "http://localhost:4318", "localhost:4318", "/v1/traces", true, "http://localhost:4318/v1/traces", false},
		{"trailing slash base", "http://collector:4318/", "collector:4318", "/v1/traces", true, "http://collector:4318/v1/traces", false},
		{"already traces path", "http://collector:4318/v1/traces", "collector:4318", "/v1/traces", true, "http://collector:4318/v1/traces", false},
		{"custom base path", "https://otlp.example.com:4318/otlp", "otlp.example.com:4318", "/otlp/v1/traces", false, "https://otlp.example.com:4318/otlp/v1/traces", false},
		{"invalid no scheme", "collector:4318", "", "", true, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hp, path, insecure, resolved, err := normalizeOTLPEndpoint(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeOTLPEndpoint() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if hp != tt.hostport {
				t.Errorf("hostport = %q, want %q", hp, tt.hostport)
			}
			if path != tt.urlPath {
				t.Errorf("urlPath = %q, want %q", path, tt.urlPath)
			}
			if insecure != tt.insecure {
				t.Errorf("insecure = %v, want %v", insecure, tt.insecure)
			}
			if resolved != tt.resolved {
				t.Errorf("resolved = %q, want %q", resolved, tt.resolved)
			}
		})
	}
}

// Test DefaultConfig function
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.True(t, config.Enabled)
	assert.Equal(t, "http://localhost:4318", config.OTLPEndpoint)
	assert.Equal(t, ServiceName, config.ServiceName)
	assert.Equal(t, ServiceVersion, config.ServiceVersion)
	assert.Equal(t, "development", config.Environment)
	assert.Equal(t, 1.0, config.SampleRate)
	assert.Equal(t, 5*time.Second, config.BatchTimeout)
	assert.Equal(t, 512, config.MaxExportBatch)
	assert.Equal(t, 2048, config.MaxQueueSize)
	assert.Equal(t, "info", config.LogLevel)
}

// Test tracer getter functions
func TestTracerGetters(t *testing.T) {
	// Test GetTracer
	tracer := GetTracer("test-tracer")
	assert.NotNil(t, tracer)

	// Test predefined tracers
	httpTracer := GetHTTPTracer()
	assert.NotNil(t, httpTracer)

	dbTracer := GetDatabaseTracer()
	assert.NotNil(t, dbTracer)

	businessTracer := GetBusinessTracer()
	assert.NotNil(t, businessTracer)

	cacheTracer := GetCacheTracer()
	assert.NotNil(t, cacheTracer)

	externalTracer := GetExternalTracer()
	assert.NotNil(t, externalTracer)
}

// Test span helper functions
func TestSpanHelpers(t *testing.T) {
	ctx := context.Background()
	tracer := GetTracer("test")

	// Test StartSpan
	newCtx, span := StartSpan(ctx, tracer, "test-span")
	assert.NotNil(t, newCtx)
	assert.NotNil(t, span)

	// Test SetSpanAttributes
	attrs := []attribute.KeyValue{
		attribute.String("test-key", "test-value"),
		attribute.Int64("test-int", 42),
	}
	SetSpanAttributes(span, attrs...)

	// Test RecordError
	testErr := assert.AnError
	RecordError(span, testErr)

	// Test SetSpanStatus
	SetSpanStatus(span, codes.Ok, "success")

	// End the span
	span.End()
}

// Test attribute helper functions
func TestAttributeHelpers(t *testing.T) {
	// Test StringAttribute
	strAttr := StringAttribute("key", "value")
	assert.Equal(t, attribute.Key("key"), strAttr.Key)
	assert.Equal(t, attribute.STRING, strAttr.Value.Type())
	assert.Equal(t, "value", strAttr.Value.AsString())

	// Test StringSliceAttribute
	sliceAttr := StringSliceAttribute("key", []string{"a", "b"})
	assert.Equal(t, attribute.Key("key"), sliceAttr.Key)
	assert.Equal(t, attribute.STRINGSLICE, sliceAttr.Value.Type())
	assert.Equal(t, []string{"a", "b"}, sliceAttr.Value.AsStringSlice())

	// Test Int64Attribute
	intAttr := Int64Attribute("key", 42)
	assert.Equal(t, attribute.Key("key"), intAttr.Key)
	assert.Equal(t, attribute.INT64, intAttr.Value.Type())
	assert.Equal(t, int64(42), intAttr.Value.AsInt64())

	// Test Float64Attribute
	floatAttr := Float64Attribute("key", 3.14)
	assert.Equal(t, attribute.Key("key"), floatAttr.Key)
	assert.Equal(t, attribute.FLOAT64, floatAttr.Value.Type())
	assert.Equal(t, 3.14, floatAttr.Value.AsFloat64())

	// Test BoolAttribute
	boolAttr := BoolAttribute("key", true)
	assert.Equal(t, attribute.Key("key"), boolAttr.Key)
	assert.Equal(t, attribute.BOOL, boolAttr.Value.Type())
	assert.Equal(t, true, boolAttr.Value.AsBool())
}

// Test Logger function
func TestLogger(t *testing.T) {
	// Test Logger function when globalLogger is nil
	logger := Logger()
	assert.NotNil(t, logger)
	assert.Equal(t, slog.Default(), logger)
}

// Test InitTelemetry with disabled config
func TestInitTelemetryDisabled(t *testing.T) {
	config := TelemetryConfig{
		Enabled: false,
	}

	err := InitTelemetry(config)
	assert.NoError(t, err)
}

// Test InitTelemetry with enabled config
func TestInitTelemetryEnabled(t *testing.T) {
	config := TelemetryConfig{
		Enabled:        true,
		OTLPEndpoint:   "http://localhost:4318",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		LogLevel:       "info",
	}

	// This will likely fail due to no OTLP endpoint, but we can test the error handling
	err := InitTelemetry(config)
	// In test environment, this might fail due to network issues or succeed
	// We're just testing that the function runs without panicking
	// and either succeeds or fails gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "failed to create OTLP exporter")
	} else {
		// If it succeeds, that's also fine
		assert.NoError(t, err)
	}
}

// Test Shutdown function
func TestShutdown(t *testing.T) {
	// Test Shutdown when globalProvider is nil
	err := Shutdown()
	assert.NoError(t, err)
}

// Test GetLogger function
func TestGetLogger(t *testing.T) {
	// First, ensure no global logger is set
	globalLogger = nil
	
	logger := GetLogger()
	// When telemetry is not initialized, this should return nil
	assert.Nil(t, logger)
	
	// Now test with a mock logger
	config := TelemetryConfig{
		Enabled: false,
	}
	err := InitTelemetry(config)
	assert.NoError(t, err)
	
	// Still should be nil when telemetry is disabled
	logger = GetLogger()
	assert.Nil(t, logger)
}

// Test InitTelemetryWithProvider with disabled config
func TestInitTelemetryWithProviderDisabled(t *testing.T) {
	config := &TelemetryConfig{
		Enabled: false,
	}
	logger := slog.Default()

	provider, err := InitTelemetryWithProvider(context.Background(), config, logger)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.NotNil(t, provider.Shutdown)
	assert.NotNil(t, provider.logger)
}

// Test InitTelemetryWithProvider with invalid endpoint
func TestInitTelemetryWithProviderInvalidEndpoint(t *testing.T) {
	config := &TelemetryConfig{
		Enabled:      true,
		OTLPEndpoint: "invalid-url://[invalid",
	}
	logger := slog.Default()

	provider, err := InitTelemetryWithProvider(context.Background(), config, logger)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "invalid OTLPEndpoint")
}
