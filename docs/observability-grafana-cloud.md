# Grafana Cloud Setup

This guide covers setting up Grafana Cloud for monitoring and observability of the SocialChef application.

## Creating Account
1. Go to [https://grafana.com/products/cloud/](https://grafana.com/products/cloud/)
2. Sign up for the free tier.
3. Create a new stack (e.g., `socialchef`).

## Enabling Application Observability
1. Navigate to your stack in the Grafana Cloud portal.
2. Go to **Configure** > **Application Observability**.
3. Choose **OpenTelemetry** as the integration method.
4. Note the **OTLP Endpoint URL** (it should look something like `https://otlp-gateway-prod-us-east-0.grafana.net/otlp`).

## Creating API Key
1. Go to **Access Policy** or **API Keys** in your stack settings.
2. Create a new access policy or API key with the following scopes/roles:
   - `metrics:write` (Metrics Publisher)
   - `logs:write` (Logs Publisher)
   - `traces:write` (Traces Publisher)
3. Generate the token and copy it. **You will not be able to see it again.**

## Configuring Production (Fly.io)
The application uses standard OpenTelemetry environment variables. Set them using the Fly.io CLI:

```bash
fly secrets set OTEL_EXPORTER_OTLP_ENDPOINT="<your-otlp-endpoint>"
fly secrets set OTEL_EXPORTER_OTLP_HEADERS="Authorization=Bearer <your-token>"
```

*Note: The `OTEL_EXPORTER_OTLP_ENDPOINT` should point to the base OTLP URL, and the application will append the appropriate paths for metrics/traces/logs if using the OTLP exporter.*

## Verifying
1. Deploy the application to production: `fly deploy`.
2. Generate some traffic by browsing the site.
3. Check **Grafana Cloud** > **Application Observability** > **Services**.
4. You should see `socialchef-remy` appearing in the list after a few minutes.
