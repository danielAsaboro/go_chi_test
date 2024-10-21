package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var tracer oteltrace.Tracer

func main() {
	// Initialize trace provider
	tp := InitTracerProvider()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()
	// Set global tracer provider & text propagators
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// Initialize tracer
	tracer = otel.Tracer("chi-server")

	// Define router
	r := chi.NewRouter()
	r.Use(otelchi.Middleware("go chi test", otelchi.WithChiRoutes(r)))

	r.HandleFunc("/users/{id:[0-9]+}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		ctx := r.Context()
		id := chi.URLParam(r, "id")
		name := getUser(ctx, id, r, startTime)

		if name == "unknown" {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		reply := fmt.Sprintf("user %s (id %s)\n", name, id)
		w.Write([]byte(reply))
	}))

	// Serve router
	log.Println("Starting server on :8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getUser(ctx context.Context, id string, r *http.Request, startTime time.Time) string {
	_, span := tracer.Start(ctx, "getUser")
	defer span.End()

	method := r.Method
	scheme := "http"
	statusCode := 200
	host := r.Host
	port := r.URL.Port()
	if port == "" {
		port = "8081"
	}

	// Set span status
	span.SetStatus(codes.Ok, "")

	// Use semantic conventions for common attributes
	span.SetAttributes(
		semconv.HTTPMethodKey.String(method),
		semconv.HTTPSchemeKey.String(scheme),
		semconv.HTTPStatusCodeKey.Int(statusCode),
		semconv.HTTPTargetKey.String(r.URL.Path),
		semconv.HTTPURLKey.String(r.URL.String()),
		semconv.HTTPHostKey.String(host),
		semconv.NetHostPortKey.String(port),
		semconv.HTTPUserAgentKey.String(r.UserAgent()),
		semconv.HTTPRequestContentLengthKey.Int64(r.ContentLength),
		semconv.NetPeerIPKey.String(r.RemoteAddr),
	)

	// Custom attributes that don't have semantic conventions
	span.SetAttributes(
		attribute.String("created_at", startTime.Format(time.RFC3339Nano)),
		attribute.Float64("duration_ns", float64(time.Since(startTime).Nanoseconds())),
		attribute.String("parent_id", ""), // You might need to extract this from the context
		attribute.String("referer", r.Referer()),
		attribute.String("request_type", "Incoming"),
		attribute.String("sdk_type", "go-chi"),
		attribute.String("service_version", ""), // Fill in your service version if available
		attribute.StringSlice("tags", []string{}),
	)

	// Set nested fields (these don't have direct semconv equivalents)
	span.SetAttributes(
		attribute.String("path_params", fmt.Sprintf(`{"id":"%s"}`, id)),
		attribute.String("query_params", fmt.Sprintf("%v", r.URL.Query())),
		attribute.String("request_body", "{}"), // Assuming empty body for GET request
		attribute.String("request_headers", fmt.Sprintf("%v", r.Header)),
		attribute.String("response_body", "{}"),
		attribute.String("response_headers", "{}"),
	)

	if id == "123" {
		return "otelchi tester"
	}
	return "unknown"
}
