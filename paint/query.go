package paint

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/NathanBaulch/rainbow-roads/conv"
	"github.com/NathanBaulch/rainbow-roads/geo"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/builtin"
	"github.com/expr-lang/expr/parser"
	"golang.org/x/exp/slices"
)

func init() {
	clear(builtin.Builtins)
	clear(builtin.Index)
	clear(builtin.Names)
	builtin.Builtins = []*builtin.Function{{Name: "is_tag"}}
	builtin.Index["is_tag"] = 0
	builtin.Names[0] = "is_tag"
}

func buildQuery(region geo.Circle, filter string) (string, error) {
	if crits, err := buildCriteria(filter); err != nil {
		return "", fmt.Errorf("overpass query error: %w", err)
	} else {
		prefix := fmt.Sprintf("way(around:%s,%s,%s)",
			conv.FormatFloat(region.Radius),
			conv.FormatFloat(region.Origin.Lat()),
			conv.FormatFloat(region.Origin.Lon()),
		)
		parts := make([]string, 0, len(crits)*3+2)
		parts = append(parts, "[out:json];(")
		for _, crit := range crits {
			parts = append(parts, prefix, crit, ";")
		}
		parts = append(parts, ");out tags geom qt;")
		return strings.Join(parts, ""), nil
	}
}

func buildCriteria(filter string) ([]string, error) {
	tree, err := parser.Parse(filter)
	if err != nil {
		return nil, err
	}

	ast.Walk(&tree.Node, &expandInArray{})
	ast.Walk(&tree.Node, &expandInRange{})
	Walk(&tree.Node, &distributeAndFoldNot{})
	toDNF(&tree.Node)

	qb := queryBuilder{}
	Walk(&tree.Node, &qb)
	if qb.err != nil {
		return nil, qb.err
	}

	for i, crit := range qb.stack {
		if !isWrapped(crit) {
			qb.stack[i] = fmt.Sprintf("(if:%s)", crit)
		}
	}

	return qb.stack, nil
}

type queryBuilder struct {
	stack []string
	not   []bool
	depth int
	err   error
}

func (q *queryBuilder) Enter(node *ast.Node) {
	if q.depth > 0 {
		q.depth++
	} else if not := asUnaryNot(*node) != nil; !not && asBinaryAnd(*node) == nil && asBinaryOr(*node) == nil {
		q.depth++
	} else {
		q.not = append(q.not, not)
	}
}

func (q *queryBuilder) Exit(node *ast.Node) {
	if q.err != nil {
		return
	}

	if q.depth > 0 {
		q.depth--
	}
	root := q.depth == 0
	not := false
	if root && len(q.not) > 0 {
		i := len(q.not) - 1
		not = q.not[i]
		q.not = q.not[:i]
	}

	if not {
		switch (*node).(type) {
		case *ast.IntegerNode, *ast.FloatNode, *ast.StringNode:
			q.err = fmt.Errorf("inverted %s not supported", nodeName(*node))
			return
		}
	}

	switch n := (*node).(type) {
	case *ast.IdentifierNode:
		name := n.Value
		if slices.IndexFunc([]rune(n.Value), func(c rune) bool { return !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') }) >= 0 {
			name = strconv.Quote(n.Value)
		}
		if !root {
			q.push(bangIf(name, not))
		} else if not {
			q.push("[", name, `="no"]`)
		} else {
			q.push("[", name, `="yes"]`)
		}
	case *ast.IntegerNode:
		q.push(n.Value)
	case *ast.FloatNode:
		q.push(n.Value)
	case *ast.BoolNode:
		if n.Value != not {
			q.push(`"yes"`)
		} else {
			q.push(`"no"`)
		}
	case *ast.StringNode:
		q.push(strconv.Quote(n.Value))
	case *ast.UnaryNode:
		if !root || (n.Operator != "not" && n.Operator != "!") {
			q.push(bangIf(n.Operator, not), "(", q.pop(), ")")
		}
	case *ast.BinaryNode:
		rhs, lhs := q.pop(), q.pop()
		switch n.Operator {
		case "and", "&&":
			if !root || (!isWrapped(lhs) && !isWrapped(rhs)) {
				q.push(lhs, "&&", rhs)
			} else {
				if !isWrapped(lhs) {
					lhs = fmt.Sprintf("(if:%s)", lhs)
				}
				if !isWrapped(rhs) {
					rhs = fmt.Sprintf("(if:%s)", rhs)
				}
				q.push(lhs, rhs)
			}
		case "or", "||":
			if !root || (!isWrapped(lhs) && !isWrapped(rhs)) {
				q.push(lhs, "||", rhs)
			} else {
				q.push(lhs)
				q.push(rhs)
			}
		case ">", ">=", "<", "<=":
			if _, ok := n.Left.(*ast.IdentifierNode); ok {
				if lhs[0] != '"' {
					lhs = strconv.Quote(lhs)
				}
				lhs = fmt.Sprintf("t[%s]", lhs)
			}
			q.push(lhs, n.Operator, rhs)
		default:
			op := n.Operator
			switch op {
			case "contains":
				op = "~"
				if _, ok := n.Right.(*ast.StringNode); ok {
					rhs = regexp.QuoteMeta(rhs)
				}
			case "startsWith":
				op = "~"
				if _, ok := n.Right.(*ast.StringNode); ok {
					rhs = rhs[:1] + "^" + regexp.QuoteMeta(rhs[1:])
				}
			case "endsWith":
				op = "~"
				if _, ok := n.Right.(*ast.StringNode); ok {
					rhs = regexp.QuoteMeta(rhs[:len(rhs)-1]) + "$" + rhs[len(rhs)-1:]
				}
			case "matches":
				op = "~"
			}
			_, okl := n.Left.(*ast.CallNode)
			_, okr := n.Right.(*ast.CallNode)
			if okl || okr {
				root = false
			}
			if root {
				if op == "==" || op == "!=" {
					if _, ok := n.Right.(*ast.IdentifierNode); ok {
						lhs, rhs = rhs, lhs
					}
					if op == "!=" {
						not = !not
					}
					if rhs == `""` {
						op = "~"
						rhs = `"^$"`
					} else {
						op = "="
					}
				}
				q.push("[", lhs, bangIf(op, not), rhs, "]")
			} else {
				q.push(lhs, bangIf(op, not), rhs)
			}
		}
	case *ast.BuiltinNode:
		q.push("[", bangIf(q.pop(), not), "]")
	case *ast.CallNode:
		name := n.Callee.String()
		parts := make([]any, 0, len(n.Arguments)+3)
		parts = append(parts, bangIf(name, not), "(")
		for range n.Arguments {
			parts = append(parts, q.pop())
		}
		parts = append(parts, ")")
		q.pop()
		q.push(parts...)
	case *ast.ConditionalNode:
		e2, e1 := q.pop(), q.pop()
		q.push(q.pop(), "?", e1, ":", e2)
	default:
		q.err = fmt.Errorf("%s not supported", nodeName(n))
	}
}

func nodeName(n ast.Node) string {
	name := reflect.TypeOf(n).Elem().Name()
	return strings.ToLower(name[:len(name)-4])
}

func isWrapped(str string) bool {
	return str[0] == '[' || strings.HasPrefix(str, "(if:")
}

func bangIf(str string, not bool) string {
	if not {
		return "!" + str
	}
	return str
}

func (q *queryBuilder) push(a ...any) {
	q.stack = append(q.stack, fmt.Sprint(a...))
}

func (q *queryBuilder) pop() string {
	i := len(q.stack) - 1
	str := q.stack[i]
	q.stack = q.stack[:i]
	return str
}
