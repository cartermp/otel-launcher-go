# Honeycomb Go Launcher

The Go Launcher is a configuration layer that chooses default values for configuration options that many OpenTelemetry users would ultimately configure manually, allowing for minimal code to quickly instrument with OpenTelemetry.

## Getting started

```bash
go get github.com/honeycombio/otel-launcher-go
```

## Configure

Minimal setup - by default will send all telemetry via GRPC to `localhost:4317`

```go
import "github.com/honeycombio/otel-launcher-go/launcher"

func main() {
    lnchr, err := launcher.ConfigureOpentelemetry()
    defer lnchr.Shutdown()
}
```

You can set headers directly instead.

```go
import "github.com/honeycombio/otel-launcher-go/launcher"

func main() {
    lnchr, err := launcher.ConfigureOpentelemetry(
        launcher.WithServiceName("service-name"),
        launcher.WithHeaders(map[string]string{
            "service-auth-key": "value",
            "service-useful-field": "testing",
        }),
    )
    defer lnchr.Shutdown()
}
```

## Configuration Options

| Config Option               | Env Variable                        | Required | Default              |
| --------------------------- | ----------------------------------- | -------- | -------------------- |
| WithServiceName             | OTEL_SERVICE_NAME                   | y        | -                    |
| WithServiceVersion          | OTEL_SERVICE_VERSION                | n        | -                    |
| WithHeaders                 | OTEL_EXPORTER_OTLP_HEADERS          | n        | {}                   |
| WithProtocol                | OTEL_EXPORTER_OTLP_PROTOCOL         | n        | grpc                 |
| WithTracesExporterEndpoint  | OTEL_EXPORTER_OTLP_TRACES_ENDPOINT  | n        | localhost:4317       |
| WithTracesExporterInsecure  | OTEL_EXPORTER_OTLP_TRACES_INSECURE  | n        | false                |
| WithMetricsExporterEndpoint | OTEL_EXPORTER_OTLP_METRICS_ENDPOINT | n        | localhost:4317       |
| WithMetricsExporterInsecure | OTEL_EXPORTER_OTLP_METRICS_INSECURE | n        | false                |
| WithLogLevel                | OTEL_LOG_LEVEL                      | n        | info                 |
| WithPropagators             | OTEL_PROPAGATORS                    | n        | tracecontext,baggage |
| WithResourceAttributes      | OTEL_RESOURCE_ATTRIBUTES            | n        | -                    |
| WithMetricsReportingPeriod  | OTEL_EXPORTER_OTLP_METRICS_PERIOD   | n        | 30s                  |
| WithMetricsEnabled          | OTEL_METRICS_ENABLED                | n        | true                 |
| WithTracesEnabled           | OTEL_TRACES_ENABLED                 | n        | true                 |

------

This is a joint effort alongside LightStep and is based their initial [otel-launcher-go](https://github.com/lightstep/otel-launcher-go). The intention is to contribute this to OpenTelemetry Go Contrib.
