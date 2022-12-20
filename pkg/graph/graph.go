package graph

import (
	"strconv"

	"github.com/kralicky/goda/pkg/pkggraph"
)

type Format interface {
	Write(*pkggraph.Graph) error
}

func PkgID(p *pkggraph.Node) string {
	// Go quoting rules are similar enough to dot quoting.
	// At least enough similar to quote a Go import path.
	return strconv.Quote(p.ID)
}
