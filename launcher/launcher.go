package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	// TODO: before merging, update to "go.opentelemetry.io/contrib/launcher".
	"github.com/honeycombio/otel-launcher-go/launcher/pipelines"
	"github.com/sethvargo/go-envconfig"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

var (
	// SetVendorOptions provides a way for a vendor to add a set of Options that are automatically applied.
	SetVendorOptions func() []Option
	// ValidateConfig is a function that a vendor can implement to provide additional validation after
	// a configuration is built.
	ValidateConfig func(*Config) error
)

// These are strings because they get appended to the host.
const (
	// GRPC default port.
	GRPCDefaultPort = "4317"
	// HTTP default port.
	HTTPDefaultPort = "4318"
)

// Option is the type of an Option to the ConfigureOpenTelemetry function; it's a
// function that accepts a config and modifies it.
type Option func(*Config)

// WithExporterEndpoint configures the generic endpoint used for sending all telemtry signals via OTLP.
func WithExporterEndpoint(url string) Option {
	return func(c *Config) {
		c.ExporterEndpoint = url
	}
}

// WithExporterInsecure permits connecting to the generic exporter endpoint without a certificate.
func WithExporterInsecure(insecure bool) Option {
	return func(c *Config) {
		c.ExporterEndpointInsecure = insecure
	}
}

// WithMetricsExporterEndpoint configures the endpoint for sending metrics via OTLP.
func WithMetricsExporterEndpoint(url string) Option {
	return func(c *Config) {
		c.MetricsExporterEndpoint = url
	}
}

// WithTracesExporterEndpoint configures the endpoint for sending traces via OTLP.
func WithTracesExporterEndpoint(url string) Option {
	return func(c *Config) {
		c.TracesExporterEndpoint = url
	}
}

// WithServiceName configures a "service.name" resource label.
func WithServiceName(name string) Option {
	return func(c *Config) {
		c.ServiceName = name
	}
}

// WithServiceVersion configures a "service.version" resource label.
func WithServiceVersion(version string) Option {
	return func(c *Config) {
		c.ServiceVersion = version
	}
}

// WithHeaders configures OTLP exporter headers.
func WithHeaders(headers map[string]string) Option {
	return func(c *Config) {
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		for k, v := range headers {
			c.Headers[k] = v
		}
	}
}

// WithTracesHeaders configures OTLP traces exporter headers.
func WithTracesHeaders(headers map[string]string) Option {
	return func(c *Config) {
		for k, v := range headers {
			c.TracesHeaders[k] = v
		}
	}
}

// WithMetricsHeaders configures OTLP metrics exporter headers.
func WithMetricsHeaders(headers map[string]string) Option {
	return func(c *Config) {
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		for k, v := range headers {
			c.MetricsHeaders[k] = v
		}
	}
}

// WithLogLevel configures the logging level for OpenTelemetry.
func WithLogLevel(loglevel string) Option {
	return func(c *Config) {
		c.LogLevel = loglevel
	}
}

// WithTracesExporterInsecure permits connecting to the
// trace endpoint without a certificate.
func WithTracesExporterInsecure(insecure bool) Option {
	return func(c *Config) {
		c.TracesExporterEndpointInsecure = insecure
	}
}

// WithMetricsExporterInsecure permits connecting to the
// metric endpoint without a certificate.
func WithMetricsExporterInsecure(insecure bool) Option {
	return func(c *Config) {
		c.MetricsExporterEndpointInsecure = insecure
	}
}

// WithResourceAttributes configures attributes on the resource; if the resource
// already exists, it sets additional attributes or overwrites those already there.
func WithResourceAttributes(attributes map[string]string) Option {
	return func(c *Config) {
		for k, v := range attributes {
			c.ResourceAttributes[k] = v
		}
	}
}

// WithPropagators configures propagators.
func WithPropagators(propagators []string) Option {
	return func(c *Config) {
		c.Propagators = propagators
	}
}

// Configures a global error handler to be used throughout an OpenTelemetry instrumented project.
// See "go.opentelemetry.io/otel".
func WithErrorHandler(handler otel.ErrorHandler) Option {
	return func(c *Config) {
		c.errorHandler = handler
	}
}

// WithMetricsReportingPeriod configures the metric reporting period,
// how often the controller collects and exports metric data.
func WithMetricsReportingPeriod(p time.Duration) Option {
	return func(c *Config) {
		c.MetricsReportingPeriod = fmt.Sprint(p)
	}
}

// WithMetricsEnabled configures whether metrics should be enabled.
func WithMetricsEnabled(enabled bool) Option {
	return func(c *Config) {
		c.MetricsEnabled = enabled
	}
}

// WithTracesEnabled configures whether traces should be enabled.
func WithTracesEnabled(enabled bool) Option {
	return func(c *Config) {
		c.TracesEnabled = enabled
	}
}

// WithSpanProcessor adds one or more SpanProcessors.
func WithSpanProcessor(sp ...trace.SpanProcessor) Option {
	return func(c *Config) {
		c.SpanProcessors = append(c.SpanProcessors, sp...)
	}
}

// WithShutdown adds functions that will be called first when the shutdown function is called.
// They are given a copy of the Config object (which has access to the Logger), and should
// return an error only in extreme circumstances, as an error return here is immediately fatal.
func WithShutdown(f func(c *Config) error) Option {
	return func(c *Config) {
		c.ShutdownFunctions = append(c.ShutdownFunctions, f)
	}
}

// Protocol defines the possible values of the protocol field.
type Protocol pipelines.Protocol

// Import the values for Protocol from pipelines but make them available without importing pipelines.
const (
	ProtocolGRPC      Protocol = Protocol(pipelines.ProtocolGRPC)
	ProtocolHTTPProto Protocol = Protocol(pipelines.ProtocolHTTPProtobuf)
	ProtocolHTTPJSON  Protocol = Protocol(pipelines.ProtocolHTTPJSON)
)

// WithExporterProtocol defines the default protocol.
func WithExporterProtocol(protocol Protocol) Option {
	return func(c *Config) {
		c.ExporterProtocol = protocol
	}
}

// WithTracesExporterProtocol defines the protocol for Traces.
func WithTracesExporterProtocol(protocol Protocol) Option {
	return func(c *Config) {
		c.TracesExporterProtocol = protocol
	}
}

// WithMetricsExporterProtocol defines the protocol for Metrics.
func WithMetricsExporterProtocol(protocol Protocol) Option {
	return func(c *Config) {
		c.MetricsExporterProtocol = protocol
	}
}

// WithSampler configures the Sampler to use when processing trace spans.
func WithSampler(sampler trace.Sampler) Option {
	return func(c *Config) {
		c.Sampler = sampler
	}
}

// Logger is an interface for a logger that can be passed to WithLogger.
type Logger interface {
	Fatalf(format string, v ...interface{})
	Debugf(format string, v ...interface{})
}

// WithLogger sets up the logger to be used by the launcher.
func WithLogger(logger Logger) Option {
	// In order to enable the environment parsing to send an error to the specified logger
	// we need to cache a copy of the logger in a package variable so that newConfig can use it
	// before we ever call the function returned by WithLogger. This is slightly messy, but
	// consistent with expected behavior of autoinstrumentation.
	defLogger = logger
	return func(c *Config) {
		c.Logger = logger
	}
}

type defaultLogger struct {
	logLevel string
}

func (l *defaultLogger) Fatalf(format string, v ...interface{}) {
	//revive:disable:deep-exit needed for default logger
	log.Fatalf(format, v...)
}

func (l *defaultLogger) Debugf(format string, v ...interface{}) {
	if l.logLevel == "debug" {
		log.Printf(format, v...)
	}
}

var defLogger Logger = &defaultLogger{logLevel: "info"}

type defaultHandler struct {
	logger Logger
}

func (l *defaultHandler) Handle(err error) {
	l.logger.Debugf("error: %v\n", err)
}

// Config is a configuration object; it is public so that it can be manipulated by vendors.
// Note that ExporterEndpoint specifies "DEFAULTPORT"; this is because the default port should
// vary depending on the protocol chosen. If not overridden by explicit configuration, it will
// be overridden with an appropriate default upon initialization.
type Config struct {
	ExporterEndpoint                string   `env:"OTEL_EXPORTER_OTLP_ENDPOINT,default=localhost"`
	ExporterEndpointInsecure        bool     `env:"OTEL_EXPORTER_OTLP_INSECURE,default=false"`
	TracesExporterEndpoint          string   `env:"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"`
	TracesExporterEndpointInsecure  bool     `env:"OTEL_EXPORTER_OTLP_TRACES_INSECURE"`
	TracesEnabled                   bool     `env:"OTEL_TRACES_ENABLED,default=true"`
	ServiceName                     string   `env:"OTEL_SERVICE_NAME"`
	ServiceVersion                  string   `env:"OTEL_SERVICE_VERSION,default=unknown"`
	MetricsExporterEndpoint         string   `env:"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"`
	MetricsExporterEndpointInsecure bool     `env:"OTEL_EXPORTER_OTLP_METRICS_INSECURE"`
	MetricsEnabled                  bool     `env:"OTEL_METRICS_ENABLED,default=true"`
	MetricsReportingPeriod          string   `env:"OTEL_EXPORTER_OTLP_METRICS_PERIOD,default=30s"`
	LogLevel                        string   `env:"OTEL_LOG_LEVEL,default=info"`
	Propagators                     []string `env:"OTEL_PROPAGATORS,default=tracecontext,baggage"`
	ResourceAttributesFromEnv       string   `env:"OTEL_RESOURCE_ATTRIBUTES"`
	ExporterProtocol                Protocol `env:"OTEL_EXPORTER_OTLP_PROTOCOL,default=grpc"`
	TracesExporterProtocol          Protocol `env:"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"`
	MetricsExporterProtocol         Protocol `env:"OTEL_EXPORTER_OTLP_METRICS_PROTOCOL"`
	Headers                         map[string]string
	TracesHeaders                   map[string]string
	MetricsHeaders                  map[string]string
	ResourceAttributes              map[string]string
	SpanProcessors                  []trace.SpanProcessor
	Sampler                         trace.Sampler
	Resource                        *resource.Resource
	Logger                          Logger                  `json:"-"`
	ShutdownFunctions               []func(c *Config) error `json:"-"`
	errorHandler                    otel.ErrorHandler
}

func newConfig(opts ...Option) *Config {
	c := &Config{
		Headers:            map[string]string{},
		TracesHeaders:      map[string]string{},
		MetricsHeaders:     map[string]string{},
		ResourceAttributes: map[string]string{},
		Logger:             defLogger,
		errorHandler:       &defaultHandler{logger: defLogger},
		Sampler:            trace.AlwaysSample(),
	}
	envError := envconfig.Process(context.Background(), c)
	if envError != nil {
		c.Logger.Fatalf("environment error: %v", envError)
	}

	// If a vendor has specific options to add, add them to opts
	vendorOpts := []Option{}
	if SetVendorOptions != nil {
		vendorOpts = append(vendorOpts, SetVendorOptions()...)
	}

	// apply vendor options then user options
	for _, opt := range append(vendorOpts, opts...) {
		opt(c)
	}

	// If using defaultLogger, update it's LogLevel to configured level
	if l, ok := c.Logger.(*defaultLogger); ok {
		l.logLevel = c.LogLevel
	}

	c.Resource = newResource(c)
	return c
}

// Launcher is the object we're here for; it implements the initialization of Open Telemetry.
type Launcher struct {
	config        *Config
	shutdownFuncs []func() error
}

func newResource(c *Config) *resource.Resource {
	r := resource.Environment()

	hostnameSet := false
	for iter := r.Iter(); iter.Next(); {
		if iter.Attribute().Key == semconv.HostNameKey && len(iter.Attribute().Value.Emit()) > 0 {
			hostnameSet = true
		}
	}

	attributes := []attribute.KeyValue{
		semconv.TelemetrySDKNameKey.String("launcher"),
		semconv.TelemetrySDKLanguageGo,
		semconv.TelemetrySDKVersionKey.String(version),
	}

	if len(c.ServiceName) > 0 {
		attributes = append(attributes, semconv.ServiceNameKey.String(c.ServiceName))
	}

	if len(c.ServiceVersion) > 0 {
		attributes = append(attributes, semconv.ServiceVersionKey.String(c.ServiceVersion))
	}

	for key, value := range c.ResourceAttributes {
		if len(value) > 0 {
			if key == string(semconv.HostNameKey) {
				hostnameSet = true
			}
			attributes = append(attributes, attribute.String(key, value))
		}
	}

	if !hostnameSet {
		hostname, err := os.Hostname()
		if err != nil {
			c.Logger.Debugf("unable to set host.name. Set OTEL_RESOURCE_ATTRIBUTES=\"host.name=<your_host_name>\" env var or configure WithResourceAttributes in code: %v", err)
		} else {
			attributes = append(attributes, semconv.HostNameKey.String(hostname))
		}
	}

	attributes = append(r.Attributes(), attributes...)

	// These detectors can't actually fail, ignoring the error.
	r, _ = resource.New(
		context.Background(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(attributes...),
	)

	// Note: There are new detectors we may wish to take advantage
	// of, now available in the default SDK (e.g., WithProcess(),
	// WithOSType(), ...).
	return r
}

type setupFunc func(*Config) (func() error, error)

// ensures that a port is set on the given host string, or adds the default port.
func ensurePort(host string, defaultPort string) string {
	ix := strings.Index(host, ":")
	switch {
	case ix < 0:
		return host + ":" + defaultPort
	case ix == len(host)-1:
		return host + defaultPort
	default:
		return host
	}
}

func (c *Config) getTracesEndpoint() (string, bool) {
	// use traces specific endpoint, falling back to generic version if not set
	if c.TracesExporterEndpoint == "" {
		// if generic endpoint is empty, traces is disabled
		if c.ExporterEndpoint == "" {
			return "", false
		}
		c.TracesExporterEndpoint = c.ExporterEndpoint
		c.TracesExporterEndpointInsecure = c.ExporterEndpointInsecure
	}

	// use traces specific protocol, falling back to generic version if not set
	if c.TracesExporterProtocol == "" {
		c.TracesExporterProtocol = c.ExporterProtocol
	}

	// use traces specific port, failling back to generic version if not set
	port := GRPCDefaultPort
	if c.TracesExporterProtocol != ProtocolGRPC {
		port = HTTPDefaultPort
	}
	return ensurePort(c.TracesExporterEndpoint, port), c.TracesExporterEndpointInsecure
}

func (c *Config) getMetricsEndpoint() (string, bool) {
	// use metrics specific endpoint, falling back to generic version if not set
	if c.MetricsExporterEndpoint == "" {
		// if generic endpoint is empty, traces is disabled
		if c.ExporterEndpoint == "" {
			return "", false
		}
		c.MetricsExporterEndpoint = c.ExporterEndpoint
		c.MetricsExporterEndpointInsecure = c.ExporterEndpointInsecure
	}

	// If a Metrics-specific protocol wasn't specified, then use the generic one,
	// which has a default value.
	if c.MetricsExporterProtocol == "" {
		c.MetricsExporterProtocol = c.ExporterProtocol
	}

	// use metrics specific port, failling back to generic version if not set
	port := HTTPDefaultPort
	if c.MetricsExporterProtocol == ProtocolGRPC {
		port = GRPCDefaultPort
	}
	return ensurePort(c.MetricsExporterEndpoint, port), c.MetricsExporterEndpointInsecure
}

func (c *Config) getTracesHeaders() map[string]string {
	// combine generic and traces headers
	headers := map[string]string{}
	for key, value := range c.Headers {
		headers[key] = value
	}
	for key, value := range c.TracesHeaders {
		headers[key] = value
	}
	return headers
}

func (c *Config) getMetricsHeaders() map[string]string {
	// combine generic and metrics headers
	headers := map[string]string{}
	for key, value := range c.Headers {
		headers[key] = value
	}
	for key, value := range c.MetricsHeaders {
		headers[key] = value
	}
	return headers
}

func setupTracing(c *Config) (func() error, error) {
	endpoint, insecure := c.getTracesEndpoint()
	if !c.TracesEnabled || endpoint == "" {
		c.Logger.Debugf("tracing is disabled by configuration: no endpoint set")
		return nil, nil
	}

	return pipelines.NewTracePipeline(pipelines.PipelineConfig{
		Protocol:       pipelines.Protocol(c.TracesExporterProtocol),
		Endpoint:       endpoint,
		Insecure:       insecure,
		Headers:        c.getTracesHeaders(),
		Resource:       c.Resource,
		Propagators:    c.Propagators,
		SpanProcessors: c.SpanProcessors,
	})
}

func setupMetrics(c *Config) (func() error, error) {
	endpoint, insecure := c.getMetricsEndpoint()
	if !c.MetricsEnabled || endpoint == "" {
		c.Logger.Debugf("metrics are disabled by configuration: no endpoint set")
		return nil, nil
	}

	return pipelines.NewMetricsPipeline(pipelines.PipelineConfig{
		Protocol:        pipelines.Protocol(c.MetricsExporterProtocol),
		Endpoint:        endpoint,
		Insecure:        insecure,
		Headers:         c.getMetricsHeaders(),
		Resource:        c.Resource,
		ReportingPeriod: c.MetricsReportingPeriod,
	})
}

// ConfigureOpenTelemetry is a function that be called with zero or more options.
// Options can be the basic ones above, or provided by individual vendors.
func ConfigureOpenTelemetry(opts ...Option) (func(), error) {
	c := newConfig(opts...)

	if c.LogLevel == "debug" {
		c.Logger.Debugf("debug logging enabled")
		c.Logger.Debugf("configuration")
		s, _ := json.MarshalIndent(c, "", "\t")
		c.Logger.Debugf(string(s))
	}

	// Give a vendor a chance to validate the configuration
	if ValidateConfig != nil {
		if err := ValidateConfig(c); err != nil {
			return nil, err
		}
	}

	if c.errorHandler != nil {
		otel.SetErrorHandler(c.errorHandler)
	}

	launcher := Launcher{
		config: c,
	}

	for _, setup := range []setupFunc{setupTracing, setupMetrics} {
		shutdown, err := setup(c)
		if err != nil {
			c.Logger.Fatalf("setup error: %v", err)
			continue
		}
		if shutdown != nil {
			launcher.shutdownFuncs = append(launcher.shutdownFuncs, shutdown)
		}
	}
	return launcher.Shutdown, nil
}

// Shutdown is the function called to shut down OpenTelemetry. It invokes any registered
// shutdown functions.
func (ls Launcher) Shutdown() {
	// call config shutdown functions first
	for _, shutdown := range ls.config.ShutdownFunctions {
		err := shutdown(ls.config)
		if err != nil {
			ls.config.Logger.Fatalf("failed to stop exporter while calling config shutdown: %v", err)
		}
	}

	for _, shutdown := range ls.shutdownFuncs {
		if err := shutdown(); err != nil {
			ls.config.Logger.Fatalf("failed to stop exporter: %v", err)
		}
	}
}
