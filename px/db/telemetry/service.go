// Package telemetry provides database telemetry and monitoring functionality.
package telemetry

import (
	"context"
	"time"

	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/px/db/conn"
)

// Attribute  span attribute.
type Attribute struct {
	Key   string
	Value any
}

// ISpan interface for span.
type ISpan interface {
	AddAttributes(attributes []Attribute)
	End()
}

// ITelemetry interface for telemetry.
type ITelemetry interface {
	// StartSpan starts new span. If returns nil, span is not created.
	StartSpan(ctx context.Context, name string) (context.Context, ISpan)
	// ObserveRequestDuration records request duration.
	ObserveRequestDuration(ctx context.Context, duration time.Duration)
	// ObserveRequest records request count.
	ObserveRequest(ctx context.Context)
	// ObserveRequestError records request error.
	ObserveRequestError(ctx context.Context, err error)
}

// Service wrapper for working with DB and sending telemetry.
type Service struct {
	parent    db.IStartStopConnector
	telemetry ITelemetry
}

// New creates a new Service instance.
func New(parent db.IStartStopConnector, telemetry ITelemetry) *Service {
	return &Service{
		parent:    parent,
		telemetry: telemetry,
	}
}

// Start starts the service.
func (s *Service) Start(ctx context.Context) error {
	return s.parent.Start(ctx)
}

// Stop stops the service.
func (s *Service) Stop(ctx context.Context) error {
	return s.parent.Stop(ctx)
}

// Connection returns an implementation of the IConnection interface.
func (s *Service) Connection(ctx context.Context, opt ...conn.ConnectionOption) conn.IConnection {
	var span ISpan
	ctx, span = s.telemetry.StartSpan(ctx, "connection")
	defer span.End()

	return newWrapper(s.parent.Connection(ctx, opt...), s.telemetryHelper)
}

func (s *Service) telemetryHelper(ctx context.Context, command, details string, arguments []any, f func() error) {
	ctxSpan, span := s.telemetry.StartSpan(ctx, "pgdb")
	if span != nil {
		ctx = ctxSpan
		defer span.End()

		attributes := make([]Attribute, 0, len(arguments)+3) //nolint:mnd // there will be 3 more
		attributes = append(attributes,
			Attribute{"command", command},
			Attribute{"query.arg.", arguments})
		if details != "" {
			attributes = append(attributes, Attribute{"details", details})
		}
		span.AddAttributes(attributes)
	}

	startTime := time.Now()

	err := f()

	s.telemetry.ObserveRequestDuration(ctx, time.Since(startTime))

	s.telemetry.ObserveRequest(ctx)
	if err != nil {
		s.telemetry.ObserveRequestError(ctx, err)
	}
}
