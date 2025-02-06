# Telemetry Package

The `telemetry` package provides database connection wrappers with support for metrics and tracing instrumentation. It allows tracking database operations with detailed telemetry data while maintaining the original connection interface.

- Automatic span creation and management for database operations
- Request duration metrics tracking
- Request count metrics
- Error tracking and reporting
- Detailed span attributes including:
  - Command type (exec, query, batch, etc.)
  - SQL query details
  - Query arguments
  - Additional operation details

```go
// Create telemetry-enabled database service
dbService := db.New(/* options */)
telemetryService := telemetry.New(dbService, myTelemetryImplementation)

// Start the service
if err := telemetryService.Start(context.Background()); err != nil {
    log.Fatal(err)
}
defer telemetryService.Stop(context.Background())

// Use the database with telemetry
connection := telemetryService.Connection(ctx)
```

The package defines the `ITelemetry` interface that must be implemented to provide telemetry functionality
