// OpenTelemetry tracing configuration for CCXT service (Bun compatible)
const { NodeSDK } = require('@opentelemetry/sdk-node');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-otlp-http');
const { ConsoleSpanExporter } = require('@opentelemetry/sdk-trace-base');
const { BatchSpanProcessor, SimpleSpanProcessor } = require('@opentelemetry/sdk-trace-base');
const { Resource } = require('@opentelemetry/resources');
const { SEMRESATTRS_SERVICE_NAME, SEMRESATTRS_SERVICE_VERSION, SEMRESATTRS_SERVICE_NAMESPACE, SEMRESATTRS_DEPLOYMENT_ENVIRONMENT, SEMRESATTRS_SERVICE_INSTANCE_ID } = require('@opentelemetry/semantic-conventions');
const { resolveOtlpTracesEndpoint } = require('./tracing-utils.js');

// Environment variables for configuration
const serviceName = process.env.OTEL_SERVICE_NAME || 'ccxt-service';
const serviceVersion = process.env.OTEL_SERVICE_VERSION || '1.0.0';
const environment = process.env.NODE_ENV || 'development';

// Robust OTLP traces endpoint resolution per OTel specs
// Prefer signal-specific env (OTEL_EXPORTER_OTLP_TRACES_ENDPOINT). If only base endpoint is provided (OTEL_EXPORTER_OTLP_ENDPOINT), ensure '/v1/traces' suffix.
const otlpTracesEnv = process.env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT;
const otlpBaseEnv = process.env.OTEL_EXPORTER_OTLP_ENDPOINT;
const resolvedOtlpEndpoint = resolveOtlpTracesEndpoint(otlpTracesEnv, otlpBaseEnv, 'http://localhost:4318');

// Create OTLP trace exporter
const otlpExporter = new OTLPTraceExporter({
  url: resolvedOtlpEndpoint,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Create console exporter for debugging
const consoleExporter = new ConsoleSpanExporter();

// Create resource with service information
const resource = new Resource({
  [SEMRESATTRS_SERVICE_NAME]: serviceName,
  [SEMRESATTRS_SERVICE_VERSION]: serviceVersion,
  [SEMRESATTRS_SERVICE_NAMESPACE]: 'celebrum-ai',
  [SEMRESATTRS_DEPLOYMENT_ENVIRONMENT]: environment,
  [SEMRESATTRS_SERVICE_INSTANCE_ID]: process.env.HOSTNAME || 'localhost',
  // Custom attributes
  'service.type': 'ccxt-exchange-api',
  'service.component': 'exchange-data-provider',
});

// Configure auto-instrumentations
const instrumentations = getNodeAutoInstrumentations({
  // Enable specific instrumentations
  '@opentelemetry/instrumentation-http': {
    enabled: true,
    requestHook: (span, request) => {
      const headers =
        typeof request.getHeaders === 'function'
          ? request.getHeaders()
          : request.headers || {};
      const apiKey =
        (typeof request.getHeader === 'function' && request.getHeader('x-api-key')) || headers['x-api-key'];

      span.setAttributes({
        'http.request.header.user_agent': headers['user-agent'] || 'unknown',
        'http.request.header.x_api_key': apiKey ? '[REDACTED]' : 'absent',
      });
    },
    responseHook: (span, response) => {
      const contentType =
        (typeof response.getHeader === 'function' && response.getHeader('content-type')) ||
        (response.headers && response.headers['content-type']) ||
        'unknown';
      span.setAttributes({
        'http.response.header.content_type': contentType,
      });
    },
  },
  '@opentelemetry/instrumentation-express': {
    enabled: true,
  },
  '@opentelemetry/instrumentation-fs': {
    enabled: true,
  },
  // Disable instrumentations that might not be relevant
  '@opentelemetry/instrumentation-dns': {
    enabled: false,
  },
  '@opentelemetry/instrumentation-net': {
    enabled: false,
  },
});

// Initialize the SDK with multiple span processors
const sdk = new NodeSDK({
  resource,
  spanProcessors: [
    new BatchSpanProcessor(otlpExporter), // OTLP exporter for SigNoz
    new SimpleSpanProcessor(consoleExporter), // Console exporter for debugging
  ],
  instrumentations,
});

// Start the SDK
sdk
  .start()
  .then(() => {
    console.log('OpenTelemetry tracing initialized successfully for CCXT service');
    console.log(`Service: ${serviceName} v${serviceVersion}`);
    if (otlpTracesEnv) {
      console.log(`OTLP Endpoint (from OTEL_EXPORTER_OTLP_TRACES_ENDPOINT): ${resolvedOtlpEndpoint}`);
    } else if (otlpBaseEnv) {
      console.log(`OTLP Endpoint (normalized from OTEL_EXPORTER_OTLP_ENDPOINT): ${resolvedOtlpEndpoint}`);
    } else {
      console.log(`OTLP Endpoint (default): ${resolvedOtlpEndpoint}`);
    }
    console.log(`Environment: ${environment}`);
  })
  .catch((error) => {
    console.error('Error initializing OpenTelemetry tracing:', error);
  });

// Graceful shutdown
process.on('SIGTERM', () => {
  sdk.shutdown()
    .then(() => console.log('OpenTelemetry terminated'))
    .catch((error) => console.error('Error terminating OpenTelemetry', error))
    .finally(() => process.exit(0));
});

process.on('SIGINT', () => {
  sdk.shutdown()
    .then(() => console.log('OpenTelemetry terminated'))
    .catch((error) => console.error('Error terminating OpenTelemetry', error))
    .finally(() => process.exit(0));
});

module.exports = sdk;
