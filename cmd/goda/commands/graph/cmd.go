package graph

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/google/subcommands"

	"github.com/kralicky/goda/pkg/graph"
	"github.com/kralicky/goda/pkg/pkggraph"
	"github.com/kralicky/goda/pkg/pkgset"
	"github.com/kralicky/goda/pkg/templates"
)

type Command struct {
	printStandard bool

	docs string

	outputType  string
	labelFormat string

	nocolor bool

	clusters bool
	shortID  bool
}

func (*Command) Name() string     { return "graph" }
func (*Command) Synopsis() string { return "Print dependency graph." }
func (*Command) Usage() string {
	return `graph <expr>:
	Print dependency dot graph.

Supported output types:

	dot - GraphViz dot format

	graphml - GraphML format

	tgf - Trivial Graph Format

	edges - format with each edge separately

	digraph - format with each node and its edges on a single line

	See "help expr" for further information about expressions.
	See "help format" for further information about formatting.
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&cmd.printStandard, "std", false, "print std packages")

	f.BoolVar(&cmd.nocolor, "nocolor", false, "disable coloring")

	f.StringVar(&cmd.docs, "docs", "https://pkg.go.dev/", "override the docs url to use")

	f.StringVar(&cmd.outputType, "type", "dot", "output type (dot, graphml, digraph, edges, tgf)")
	f.StringVar(&cmd.labelFormat, "f", "", "label formatting")

	f.BoolVar(&cmd.clusters, "cluster", false, "create clusters")
	f.BoolVar(&cmd.shortID, "short", false, "use short package id-s inside clusters")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if cmd.labelFormat == "" {
		switch cmd.outputType {
		case "dot":
			cmd.labelFormat = `{{.ID}}\l{{ .Stat.Go.Lines }} / {{ .Stat.Go.Size }}\l`
		default:
			cmd.labelFormat = `{{.ID}}`
		}
	}

	label, err := templates.Parse(cmd.labelFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid label format: %v\n", err)
		return subcommands.ExitFailure
	}

	var format graph.Format
	switch strings.ToLower(cmd.outputType) {
	case "dot":
		format = &graph.Dot{
			Out:      os.Stdout,
			Err:      os.Stderr,
			Docs:     cmd.docs,
			Clusters: cmd.clusters,
			NoColor:  cmd.nocolor,
			ShortID:  cmd.shortID,
			Tmpl:     label,
		}
	case "digraph":
		format = &graph.Digraph{
			Out:  os.Stdout,
			Err:  os.Stderr,
			Tmpl: label,
		}
	case "tgf":
		format = &graph.TGF{
			Out:  os.Stdout,
			Err:  os.Stderr,
			Tmpl: label,
		}
	case "edges":
		format = &graph.Edges{
			Out:  os.Stdout,
			Err:  os.Stderr,
			Tmpl: label,
		}
	case "graphml":
		format = &graph.GraphML{
			Out:  os.Stdout,
			Err:  os.Stderr,
			Tmpl: label,
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown output type %q\n", cmd.outputType)
		return subcommands.ExitFailure
	}

	if !cmd.printStandard {
		go pkgset.LoadStd()
	}

	result, err := pkgset.Calc(ctx, f.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return subcommands.ExitFailure
	}
	if !cmd.printStandard {
		result = pkgset.Subtract(result, pkgset.Std())
	}

	graph := pkggraph.From(result)
	if err := format.Write(graph); err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
