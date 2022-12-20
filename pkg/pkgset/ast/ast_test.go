package ast

import (
	"reflect"
	"testing"
)

func TestParsing(t *testing.T) {
	tests := []struct {
		input  string
		clean  string
		tokens []Token
	}{{
		"", "", nil,
	}, {
		"golang.org/x/tools/...",
		"golang.org/x/tools/...",
		[]Token{
			{TPackage, "golang.org/x/tools/..."},
		},
	}, {
		"  github.com/kralicky/goda    golang.org/x/tools/...  ",
		"(github.com/kralicky/goda, golang.org/x/tools/...)",
		[]Token{
			{TPackage, "github.com/kralicky/goda"},
			{TPackage, "golang.org/x/tools/..."},
		},
	}, {
		"  github.com/kralicky/goda  +  golang.org/x/tools/...  ",
		"+(github.com/kralicky/goda, golang.org/x/tools/...)",
		[]Token{
			{TPackage, "github.com/kralicky/goda"},
			{TOp, "+"},
			{TPackage, "golang.org/x/tools/..."},
		},
	}, {
		"std - (std - unsafe:all)",
		"-(std, -(std, unsafe:all))",
		[]Token{
			{TPackage, "std"},
			{TOp, "-"},
			{TLeftParen, "("},
			{TPackage, "std"},
			{TOp, "-"},
			{TPackage, "unsafe"},
			{TSelector, "all"},
			{TRightParen, ")"},
		},
	}, {
		"  github.com/kralicky/goda:all - golang.org/x/tools/...  ",
		"-(github.com/kralicky/goda:all, golang.org/x/tools/...)",
		[]Token{
			{TPackage, "github.com/kralicky/goda"},
			{TSelector, "all"},
			{TOp, "-"},
			{TPackage, "golang.org/x/tools/..."},
		},
	}, {
		"Reaches(github.com/kralicky/goda +   github.com/loov/qloc, golang.org/x/tools/...:all)",
		"Reaches(+(github.com/kralicky/goda, github.com/loov/qloc), golang.org/x/tools/...:all)",
		[]Token{
			{TFunc, "Reaches"},
			{TLeftParen, "("},
			{TPackage, "github.com/kralicky/goda"},
			{TOp, "+"},
			{TPackage, "github.com/loov/qloc"},
			{TComma, ","},
			{TPackage, "golang.org/x/tools/..."},
			{TSelector, "all"},
			{TRightParen, ")"},
		},
	}, {
		"Reaches(github.com/kralicky/goda, golang.org/x/tools/...:all):import:all",
		"Reaches(github.com/kralicky/goda, golang.org/x/tools/...:all):import:all",
		[]Token{
			{TFunc, "Reaches"},
			{TLeftParen, "("},
			{TPackage, "github.com/kralicky/goda"},
			{TComma, ","},
			{TPackage, "golang.org/x/tools/..."},
			{TSelector, "all"},
			{TRightParen, ")"},
			{TSelector, "import"},
			{TSelector, "all"},
		},
	}, {
		"test=1(github.com/kralicky/goda)",
		"test=1(github.com/kralicky/goda)",
		[]Token{
			{TFunc, "test=1"},
			{TLeftParen, "("},
			{TPackage, "github.com/kralicky/goda"},
			{TRightParen, ")"},
		},
	}, {
		"test=1(github.com/kralicky/goda) - test=0(github.com/kralicky/goda)",
		"-(test=1(github.com/kralicky/goda), test=0(github.com/kralicky/goda))",
		[]Token{
			{TFunc, "test=1"},
			{TLeftParen, "("},
			{TPackage, "github.com/kralicky/goda"},
			{TRightParen, ")"},
			{TOp, "-"},
			{TFunc, "test=0"},
			{TLeftParen, "("},
			{TPackage, "github.com/kralicky/goda"},
			{TRightParen, ")"},
		},
	}, {
		"x:-test:+test",
		"x:-test:+test",
		[]Token{
			{TPackage, "x"},
			{TSelector, "-test"},
			{TSelector, "+test"},
		},
	}, {
		"(x + y):+test",
		"+(x, y):+test",
		[]Token{
			{TLeftParen, "("},
			{TPackage, "x"},
			{TOp, "+"},
			{TPackage, "y"},
			{TRightParen, ")"},
			{TSelector, "+test"},
		},
	}}

	for _, test := range tests {
		tokens, err := Tokenize(test.input)
		if err != nil {
			t.Errorf("\nlex %q\n\tgot:%v\n\terr:%v", test.input, tokens, err)
			continue
		}
		if len(tokens) == 0 {
			tokens = nil
		}

		if !reflect.DeepEqual(tokens, test.tokens) {
			t.Errorf("\nlex %q\n\texp:%v\n\tgot:%v", test.input, test.tokens, tokens)
			continue
		}

		expr, err := Parse(tokens)
		if err != nil {
			t.Errorf("\nparse %q\n\terr:%v", test.input, err)
			continue
		}
		if expr == nil {
			continue
		}

		clean := expr.String()
		if clean != test.clean {
			t.Errorf("\nparse %q\n\texp:%v\n\tgot:%v", test.input, test.clean, clean)
			t.Log("\nTREE\n", expr.Tree(0))
			continue
		}
	}
}
