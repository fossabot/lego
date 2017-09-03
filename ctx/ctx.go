package ctx

import (
	netCtx "golang.org/x/net/context"

	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/stats"
)

// Ctx is the root interface that defines a context
type Ctx interface {
	netCtx.Context

	Logger
	Stats
}

// Logger provides the core interface for logging
type Logger interface {
	Trace(tag, msg string, fields ...log.Field)
	Warning(tag, msg string, fields ...log.Field)
	Error(tag, msg string, fields ...log.Field)
}

// Stats provides the core interface for stats
type Stats interface {
	Stats() stats.Stats
}
