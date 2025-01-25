package db

import (
	"context"

	"github.com/n-r-w/pgh/v2/px/db/conn"
)

//go:generate mockgen -source interface.go -destination interface_mock.go -package db

// IConnectionGetter interface for getting connections.
// Created for ease of use of this package in other projects.
type IConnectionGetter interface {
	Connection(ctx context.Context, opt ...conn.ConnectionOption) conn.IConnection
}

// IStartStopConnector - interface for a service that creates IConnection and can be started and stopped.
type IStartStopConnector interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Connection(ctx context.Context, opt ...conn.ConnectionOption) conn.IConnection
}
