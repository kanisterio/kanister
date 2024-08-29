package tracing

import (
	"context"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// func Init(service string) (opentracing.Tracer, io.Closer) {
// 	cfg := &config.Configuration{
// 		ServiceName: service,
// 		Sampler: &config.SamplerConfig{
// 			Type:  "const",
// 			Param: 1,
// 		},
// 		Reporter: &config.ReporterConfig{
// 			LogSpans: true,
// 		},
// 	}
// 	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
// 	if err != nil {
// 		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
// 	}
// 	return tracer, closer
// }

func Init(ctx context.Context) (*trace.TracerProvider, error) {
	// Create the OTLP exporter.
	se := otlptracegrpc.NewUnstarted()

	r := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("kanister"),
		semconv.ServiceVersionKey.String("0.0.1"),
	)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(se),
		trace.WithResource(r),
	)

	if err := se.Start(ctx); err != nil {
		return nil, errors.Wrap(err, "unable to start OTLP exporter")
	}

	otel.SetTracerProvider(tp)
	return tp, nil
}
