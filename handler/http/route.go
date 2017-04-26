package http

import (
	"bytes"
	"fmt"
	"text/tabwriter"
)

// Route is an action attached to a given HTTP endpoint (method + path)
type Route struct {
	Path   string
	Method string
	Action ActionFunc
}

// table returns a nice-looking route table
func table(routes []Route) string {
	// Tabwriter to display nice route table
	w := new(tabwriter.Writer)
	buf := bytes.NewBuffer([]byte{})
	w.Init(buf, 0, 8, 0, '\t', 0)
	fmt.Fprintln(w, "Method\tPath\tAction")

	for _, route := range routes {
		fmt.Fprintf(w, "%s\t%s\t%T\n", route.Method, route.Path, route.Action)
	}

	w.Flush()
	return buf.String()
}
