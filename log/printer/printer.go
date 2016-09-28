package printer

import (
	"sort"
	"sync"

	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/log/printer/stdout"
)

func init() {
	Register(stdout.Name, stdout.New)
}

// Printer returns a new logger initialised with the given config
type Printer func(config map[string]string) (log.Printer, error)

func New(config *config.Log) (log.Printer, error) {
	return newLogger(config.Printer.Adapter, config.Printer.Config)
}

var (
	printersMu sync.RWMutex
	printers   = make(map[string]Printer)
)

// Printers returns the list of registered printers
func Printers() []string {
	printersMu.RLock()
	defer printersMu.RUnlock()

	var l []string
	for a := range printers {
		l = append(l, a)
	}

	sort.Strings(l)

	return l
}

// Register makes a logger printer available by the provided name.
// If an printer is registered twice or if an printer is nil, it will panic.
func Register(name string, printer Printer) {
	printersMu.Lock()
	defer printersMu.Unlock()

	if printer == nil {
		panic("logs: Registered printer is nil")
	}
	if _, dup := printers[name]; dup {
		panic("logs: Duplicated printer")
	}

	printers[name] = printer
}

// newLogger returns a new logger instance
func newLogger(printer string, config map[string]string) (log.Printer, error) {
	printersMu.RLock()
	defer printersMu.RUnlock()

	if f, ok := printers[printer]; ok {
		return f(config)
	}

	return stdout.New(config)
}
