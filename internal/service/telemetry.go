package service

import (
	"context"
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/zjregee/alter/internal/utils"
)

var (
	telemetryInitOnce sync.Once
	tracerProvider    *sdktrace.TracerProvider
	telemetryEnabled  bool
)

func ensureTelemetry(ctx context.Context) {
	telemetryInitOnce.Do(func() {
		if !shouldInitTelemetry() {
			utils.GetLogger().Printf("telemetry disabled: no OTEL or Langfuse configuration found")
			return
		}

		exporter, err := otlptracehttp.New(context.Background())
		if err != nil {
			utils.GetLogger().Printf("telemetry init failed (exporter): %v", err)
			return
		}

		serviceName := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME"))
		if serviceName == "" {
			serviceName = "alter"
		}

		res, err := resource.New(ctx,
			resource.WithAttributes(
				semconv.ServiceNameKey.String(serviceName),
				attribute.String("telemetry.sdk.language", "go"),
			),
		)
		if err != nil {
			utils.GetLogger().Printf("telemetry init failed (resource): %v", err)
			return
		}

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		tracerProvider = tp
		otel.SetTracerProvider(tp)
		telemetryEnabled = true
	})
}

func shouldInitTelemetry() bool {
	return strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")) != "" ||
		strings.TrimSpace(os.Getenv("LANGFUSE_PUBLIC_KEY")) != "" ||
		strings.TrimSpace(os.Getenv("LANGFUSE_SECRET_KEY")) != ""
}

func ShutdownTelemetry(ctx context.Context) error {
	if tracerProvider != nil {
		return tracerProvider.Shutdown(ctx)
	}
	return nil
}

func isTelemetryEnabled() bool {
	return telemetryEnabled && tracerProvider != nil
}
