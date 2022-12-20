package graph

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/kralicky/goda/pkg/pkggraph"
)

type TGF struct {
	Out  io.Writer
	Err  io.Writer
	Tmpl *template.Template
}

func (ctx *TGF) Label(p *pkggraph.Node) string {
	var labelText strings.Builder
	err := ctx.Tmpl.Execute(&labelText, p)
	if err != nil {
		fmt.Fprintf(ctx.Err, "template error: %v\n", err)
	}
	return labelText.String()
}

func (ctx *TGF) Write(graph *pkggraph.Graph) error {
	indexCache := map[*pkggraph.Node]int64{}
	for i, node := range graph.Sorted {
		label := ctx.Label(node)
		indexCache[node] = int64(i + 1)
		fmt.Fprintf(ctx.Out, "%d %s\n", i+1, label)
	}

	fmt.Fprintf(ctx.Out, "#\n")

	for _, node := range graph.Sorted {
		for _, imp := range node.ImportsNodes {
			fmt.Fprintf(ctx.Out, "%d %d\n", indexCache[node], indexCache[imp])
		}
	}

	return nil
}
