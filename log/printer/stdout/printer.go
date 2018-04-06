// Package stdout prints log lines into the standard output.
// It also colorised outputs with ANSI Escape Codes
package stdout

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/log"
)

const Name = "stdout"

var (
	traceColour   = color.New(color.FgBlue)
	warningColour = color.New(color.FgYellow)
	errorColour   = color.New(color.FgRed)
	unknownColour = color.New(color.FgWhite)
)

func New(c config.Tree) (log.Printer, error) {
	return &Logger{}, nil
}

type Logger struct{}

func (l *Logger) Print(ctx *log.Ctx, s string) error {
	colour := pickColour(ctx.Level)
	fmt.Println(colour.SprintFunc()(s))
	return nil
}

func pickColour(lvl string) *color.Color {
	switch lvl {
	case "TR":
		return traceColour
	case "WN":
		return warningColour
	case "ER":
		return errorColour
	}

	return unknownColour
}
