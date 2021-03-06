# health-exporter

Health Exporter for [Prometheus](http://prometheus.io/)

## Deprecation

**The health exporter won't be actively maintained anymore**, since the official [blackbox exporter](https://github.com/prometheus/blackbox_exporter) solves the same use case - and even more.

## About

Checks a list of service URLs for their HTTP response status code. Each service status will be exposed as Prometheus
metric, additionally an overall result will be exposed as `health_overall`.

The default `config.json` should give you an idea how to configure your services. You only need to define the
health check URL and optionally some labels you want to appear in your metrics. Please note that labels
declared for one service but not for another service, will have empty values.

## Usage

```
$ docker run -d -p 9990:9990 -v `pwd`/config.json:/config.json gesellix/health-exporter -config.file=/config.json
```

By default the `config.json` is read from the current directory, but you can use another config
with the `-config.file=...` command line argument. The *health-exporter* listens on port `9990` by default
and exposes all metrics under the `/metrics` path. Use `./health-exporter --help` to see all options.

## Example

The following example config would result in the metrics shown below.
The `requestTimeoutMillis` property defaults to _500_ when missing:

Config:

```
{
  "requestTimeoutMillis": 500,
  "services": [
    {
      "uri": "http://localhost:8080/health",
      "labels": {
        "name": "test",
        "stage": "dev",
        "foo": "bar"
      }
    },
    {
      "uri": "http://localhost:8090/another/health",
      "labels": {
        "name": "another",
        "stage": "qa",
        "bar": "baz"
      }
    }
  ]
}
```

Metrics:

```
# HELP health_overall overall service availability
# TYPE health_overall gauge
health_overall 0
# HELP health_service service status summary
# TYPE health_service gauge
health_service{bar="",foo="bar",name="test",stage="dev",uri="http://localhost:8080/health"} 1
health_service{bar="baz",foo="",name="another",stage="qa",uri="http://localhost:8090/another/health"} 0
```
