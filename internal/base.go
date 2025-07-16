package internal

import "context"

type BaseApp interface {
	Startup(ctx context.Context)
	Shutdown(ctx context.Context)
}
