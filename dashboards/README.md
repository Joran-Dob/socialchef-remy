# SocialChef Grafana Dashboards

This directory contains Grafana dashboard JSON files that can be imported into your Grafana instance to monitor the SocialChef Remy application.

## Prerequisites

- A Grafana instance (v10.x or later recommended).
- A Prometheus datasource configured to receive **Span Metrics** from Tempo or an OpenTelemetry Collector.
  - The dashboards use the `traces_span_metrics_*` metric names.
  - Metrics should include labels like `service_name`, `span_name`, `http_target`, `db_system`, and `db_operation`.

## Available Dashboards

1.  **HTTP Overview** (`http-overview.json`):
    - Request Rate (req/s)
    - Latency (p50, p95, p99)
    - Error Rate (%)
    - Top Routes by traffic
2.  **Worker Overview** (`worker-overview.json`):
    - Job Processing Rate by job type
    - Job Duration (p95)
    - Error Rate by job type
3.  **Database Overview** (`database-overview.json`):
    - Query Duration (p95) by operation
    - Query Rate by operation
4.  **Reference Metrics** (`http-metrics.json`, `worker-metrics.json`, `business-metrics.json`):
    - Documentation and reference PromQL queries for Better Stack or custom Prometheus dashboards.
    - Based on explicit OTel metrics rather than span metrics.

## How to Import

1.  Open your Grafana instance.
2.  In the left sidebar, go to **Dashboards** > **Import**.
3.  Either:
    - Click **Upload JSON file** and select one of the `.json` files in this directory.
    - Copy the contents of a JSON file and paste it into the **Import via panel json** text area.
4.  Click **Load**.
5.  Select your Prometheus datasource from the dropdown (the dashboards use a `${datasource}` template variable).
6.  Click **Import**.

## Customization

- **Service Name**: The dashboards are configured to filter for `service_name="socialchef-server"`. If you renamed your service, you may need to update the queries in the JSON files.
- **Refresh Rate**: Default refresh rate is set to 5s.
- **Time Range**: Default time range is the last 1 hour.
