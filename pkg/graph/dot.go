package graph

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/kralicky/goda/pkg/pkggraph"
	"github.com/kralicky/goda/pkg/pkgtree"
)

type Dot struct {
	Out io.Writer
	Err io.Writer

	Docs     string
	Clusters bool
	NoColor  bool
	ShortID  bool

	Tmpl *template.Template
}

func (ctx *Dot) Label(p *pkggraph.Node) string {
	var labelText strings.Builder
	err := ctx.Tmpl.Execute(&labelText, p)
	if err != nil {
		fmt.Fprintf(ctx.Err, "template error: %v\n", err)
	}
	return labelText.String()
}

func (ctx *Dot) ModuleLabel(mod *pkgtree.Module) string {
	lbl := mod.Mod.Path
	if mod.Mod.Version != "" {
		lbl += "@" + mod.Mod.Version
	}
	if mod.Local {
		lbl += " (local)"
	}
	if rep := mod.Mod.Replace; rep != nil {
		lbl += " =>\\n" + rep.Path
		if rep.Version != "" {
			lbl += "@" + rep.Version
		}
	}
	return lbl
}

func (ctx *Dot) TreePackageLabel(tp *pkgtree.Package, parentPrinted bool) string {
	suffix := ""
	parentPath := tp.Parent.Path()
	if parentPrinted && tp.Parent != nil && parentPath != "" {
		suffix = strings.TrimPrefix(tp.Path(), parentPath+"/")
	}

	if suffix != "" && ctx.ShortID {
		defer func(previousID string) { tp.GraphNode.ID = previousID }(tp.GraphNode.ID)
		tp.GraphNode.ID = suffix
	}

	var labelText strings.Builder
	err := ctx.Tmpl.Execute(&labelText, tp.GraphNode)
	if err != nil {
		fmt.Fprintf(ctx.Err, "template error: %v\n", err)
	}
	return labelText.String()
}

func (ctx *Dot) RepoRef(repo *pkgtree.Repo) string {
	return fmt.Sprintf(`href=%q`, ctx.Docs+repo.Path())
}

func (ctx *Dot) ModuleRef(mod *pkgtree.Module) string {
	return fmt.Sprintf(`href=%q`, ctx.Docs+mod.Path()+"@"+mod.Mod.Version)
}

func (ctx *Dot) TreePackageRef(tp *pkgtree.Package) string {
	return fmt.Sprintf(`href=%q`, ctx.Docs+tp.Path())
}

func (ctx *Dot) Ref(p *pkggraph.Node) string {
	return fmt.Sprintf(`href=%q`, ctx.Docs+p.ID)
}

func (ctx *Dot) writeGraphProperties() {
	if ctx.NoColor {
		fmt.Fprintf(ctx.Out, "    node [fontsize=10 shape=rectangle target=\"_graphviz\"];\n")
		fmt.Fprintf(ctx.Out, "    edge [tailport=e];\n")
	} else {
		fmt.Fprintf(ctx.Out, "    node [penwidth=2 fontsize=10 shape=rectangle target=\"_graphviz\"];\n")
		fmt.Fprintf(ctx.Out, "    edge [tailport=e penwidth=2];\n")
	}
	fmt.Fprintf(ctx.Out, "    compound=true;\n")

	fmt.Fprintf(ctx.Out, "    rankdir=LR;\n")
	fmt.Fprintf(ctx.Out, "    newrank=true;\n")
	fmt.Fprintf(ctx.Out, "    ranksep=\"1.5\";\n")
	fmt.Fprintf(ctx.Out, "    quantum=\"0.5\";\n")
}

func (ctx *Dot) Write(graph *pkggraph.Graph) error {
	if ctx.Clusters {
		return ctx.WriteClusters(graph)
	} else {
		return ctx.WriteRegular(graph)
	}
}

func (ctx *Dot) WriteRegular(graph *pkggraph.Graph) error {
	fmt.Fprintf(ctx.Out, "digraph G {\n")
	ctx.writeGraphProperties()
	defer fmt.Fprintf(ctx.Out, "}\n")

	for _, n := range graph.Sorted {
		fmt.Fprintf(ctx.Out, "    %v [label=\"%v\" %v %v];\n", PkgID(n), ctx.Label(n), ctx.Ref(n), ctx.colorOf(n))
	}

	for _, src := range graph.Sorted {
		for _, dst := range src.ImportsNodes {
			fmt.Fprintf(ctx.Out, "    %v -> %v [%v];\n", PkgID(src), PkgID(dst), ctx.colorOf(dst))
		}
	}

	return nil
}

func (ctx *Dot) WriteClusters(graph *pkggraph.Graph) error {
	root, err := pkgtree.From(graph)
	if err != nil {
		return fmt.Errorf("failed to construct cluster tree: %v", err)
	}
	lookup := root.LookupTable()
	isCluster := map[*pkggraph.Node]bool{}

	fmt.Fprintf(ctx.Out, "digraph G {\n")
	ctx.writeGraphProperties()
	defer fmt.Fprintf(ctx.Out, "}\n")

	printed := make(map[pkgtree.Node]bool)

	var visit func(tn pkgtree.Node)
	visit = func(tn pkgtree.Node) {
		switch tn := tn.(type) {
		case *pkgtree.Repo:
			if tn.SameAsOnlyModule() {
				break
			}
			printed[tn] = true
			fmt.Fprintf(ctx.Out, "subgraph %q {\n", "cluster_"+tn.Path())
			fmt.Fprintf(ctx.Out, "    label=\"%v\"\n", tn.Path())
			fmt.Fprintf(ctx.Out, "    tooltip=\"%v\"\n", tn.Path())
			fmt.Fprintf(ctx.Out, "    %v\n", ctx.RepoRef(tn))
			defer fmt.Fprintf(ctx.Out, "}\n")

		case *pkgtree.Module:
			printed[tn] = true
			label := ctx.ModuleLabel(tn)
			fmt.Fprintf(ctx.Out, "subgraph %q {\n", "cluster_"+tn.Path())
			fmt.Fprintf(ctx.Out, "    label=\"%v\"\n", label)
			fmt.Fprintf(ctx.Out, "    tooltip=\"%v\"\n", label)
			fmt.Fprintf(ctx.Out, "    %v\n", ctx.ModuleRef(tn))
			defer fmt.Fprintf(ctx.Out, "}\n")

		case *pkgtree.Package:
			printed[tn] = true
			gn := tn.GraphNode
			if tn.Path() == tn.Parent.Path() {
				isCluster[tn.GraphNode] = true
				shape := "circle"
				if tn.OnlyChild() {
					shape = "point"
				}
				fmt.Fprintf(ctx.Out, "    %v [label=\"\" tooltip=\"%v\" shape=%v %v rank=0];\n", PkgID(gn), tn.Path(), shape, ctx.colorOf(gn))
			} else {
				label := ctx.TreePackageLabel(tn, printed[tn.Parent])
				href := ctx.TreePackageRef(tn)
				fmt.Fprintf(ctx.Out, "    %v [label=\"%v\" tooltip=\"%v\" %v %v];\n", PkgID(gn), label, tn.Path(), href, ctx.colorOf(gn))
			}
		}

		tn.VisitChildren(visit)
	}
	root.VisitChildren(visit)

	for _, src := range graph.Sorted {
		srctree := lookup[src]
		for _, dst := range src.ImportsNodes {
			dstID := PkgID(dst)
			dstTree := lookup[dst]
			tooltip := src.ID + " -> " + dst.ID

			if isCluster[dst] && srctree.Parent != dstTree {
				fmt.Fprintf(ctx.Out, "    %v -> %v [tooltip=\"%v\" lhead=%q %v];\n", PkgID(src), dstID, tooltip, "cluster_"+dst.ID, ctx.colorOf(dst))
			} else {
				fmt.Fprintf(ctx.Out, "    %v -> %v [tooltip=\"%v\" %v];\n", PkgID(src), dstID, tooltip, ctx.colorOf(dst))
			}
		}
	}

	return nil
}

func (ctx *Dot) colorOf(p *pkggraph.Node) string {
	if ctx.NoColor {
		return ""
	}

	hash := sha256.Sum256([]byte(p.PkgPath))
	hue := float64(uint(hash[0])<<8|uint(hash[1])) / 0xFFFF
	return "color=" + hslahex(hue, 0.9, 0.3, 0.7)
}
