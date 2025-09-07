package internal

import "context"

type BaseApp interface {
	OnStartup(ctx context.Context)
	Shutdown(ctx context.Context)
}
