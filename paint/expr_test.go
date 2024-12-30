package paint

import (
	"fmt"
	"testing"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/stretchr/testify/require"
)

func TestExpandInArray(t *testing.T) {
	testCases := []struct{ input, want string }{
		{"A in []", "false"},
		{"A in [B]", "A == B"},
		{"A in [B,C]", "A == B or A == C"},
		{"A not in []", "not(false)"},
		{"A not in [B]", "not(A == B)"},
		{"A not in [B,C]", "not(A == B or A == C)"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			gotT, err := parser.Parse(tc.input)
			is.NoError(err)
			ast.Walk(&gotT.Node, &expandInArray{})
			wantT, err := parser.Parse(tc.want)
			is.NoError(err)

			is.Equal(ast.Dump(wantT.Node), ast.Dump(gotT.Node))
		})
	}
}

func TestExpandInRange(t *testing.T) {
	testCases := []struct{ input, want string }{
		{"A in 2..2", "A == 2"},
		{"A in 2..4", "A >= 2 and A <= 4"},
		{"A not in 2..4", "not(A >= 2 and A <= 4)"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			gotT, err := parser.Parse(tc.input)
			is.NoError(err)
			ast.Walk(&gotT.Node, &expandInRange{})
			wantT, err := parser.Parse(tc.want)
			is.NoError(err)

			is.Equal(ast.Dump(wantT.Node), ast.Dump(gotT.Node))
		})
	}
}

func TestDistributeAndFoldNot(t *testing.T) {
	testCases := []struct{ input, want string }{
		{"!!A", "A"},
		{"!!!A", "!A"},
		{"!!!!A", "A"},
		{"!true", "false"},
		{"!not(!A)", "!A"},
		{"!(A or B)", "!A and !B"},
		{"!(A and B)", "!A or !B"},
		{"!(!(A && B) || !C)", "A && B && C"},
		{"!(A == B)", "A != B"},
		{"!(A > B)", "A <= B"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			gotT, err := parser.Parse(tc.input)
			is.NoError(err)
			Walk(&gotT.Node, &distributeAndFoldNot{})
			wantT, err := parser.Parse(tc.want)
			is.NoError(err)

			is.Equal(ast.Dump(wantT.Node), ast.Dump(gotT.Node))
		})
	}
}

func TestDNF(t *testing.T) {
	testCases := []struct{ input, want string }{
		{"A and (B or C)", "(A and B) or (A and C)"},
		{"(A or B) and C", "(A and C) or (B and C)"},
		{"(A or B or C) and D", "(A and D) or (B and D) or (C and D)"},
		{"(A or B) and (C or D)", "(A and C) or (A and D) or ((B and C) or (B and D))"},
		{"(A and (B or C)) or D", "(A and B) or (A and C) or D"},
		{"((A or B) and C) or D", "(A and C) or (B and C) or D"},
		{"((A or B or C) and D) or E", "(A and D) or (B and D) or (C and D) or E"},
		{"((A or B) and (C or D)) or E", "(A and C) or (A and D) or ((B and C) or (B and D)) or E"},
		{"(A and (B or C)) and D", "(A and B and D) or (A and C and D)"},
		{"(A or B) and C and D", "(A and C and D) or (B and C and D)"},
		{"(A or B or C) and D and E", "(A and D and E) or (B and D and E) or (C and D and E)"},
		{"(A or B) and (C or D) and E", "(A and C and E) or (A and D and E) or ((B and C and E) or (B and D and E))"},
		{"A and (B or foo(C and (D or E)))", "(A and B) or (A and foo(C and (D or E)))"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			gotT, err := parser.Parse(tc.input)
			is.NoError(err)
			toDNF(&gotT.Node)
			wantT, err := parser.Parse(tc.want)
			is.NoError(err)

			is.Equal(ast.Dump(wantT.Node), ast.Dump(gotT.Node))
		})
	}
}
