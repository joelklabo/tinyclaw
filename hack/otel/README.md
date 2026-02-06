# Local Telemetry Stack

Local OpenTelemetry + Grafana stack for tinyclaw development.

## Quick Start

```bash
cd hack/otel
docker compose up -d
```

## Endpoints

| Service       | URL                    | Description          |
|---------------|------------------------|----------------------|
| OTLP gRPC     | localhost:4317         | Traces & metrics     |
| OTLP HTTP     | localhost:4318         | Traces & metrics     |
| Grafana       | http://localhost:3000  | Dashboards & explore |

## Configuration

Set the OTLP endpoint for tinyclaw:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

## Teardown

```bash
docker compose down
```
