package graph

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/kralicky/goda/pkg/pkggraph"
)

type Digraph struct {
	Out  io.Writer
	Err  io.Writer
	Tmpl *template.Template
}

func (ctx *Digraph) Label(p *pkggraph.Node) string {
	var labelText strings.Builder
	err := ctx.Tmpl.Execute(&labelText, p)
	if err != nil {
		fmt.Fprintf(ctx.Err, "template error: %v\n", err)
	}
	return labelText.String()
}

func (ctx *Digraph) Write(graph *pkggraph.Graph) error {
	labelCache := map[*pkggraph.Node]string{}
	for _, node := range graph.Sorted {
		labelCache[node] = ctx.Label(node)
	}
	for _, node := range graph.Sorted {
		fmt.Fprintf(ctx.Out, "%s", labelCache[node])
		for _, imp := range node.ImportsNodes {
			fmt.Fprintf(ctx.Out, " %s", labelCache[imp])
		}
		fmt.Fprintf(ctx.Out, "\n")
	}

	return nil
}
