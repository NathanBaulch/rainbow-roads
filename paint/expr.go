package paint

import (
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/ast"
	"github.com/antonmedv/expr/vm"
)

var operatorPairs = map[string]string{
	"and": "or",
	"&&":  "||",
	"==":  "!=",
	">=":  "<",
	">":   "<=",
	"in":  "not in",
}

func init() {
	for k, v := range operatorPairs {
		operatorPairs[v] = k
	}
}

func mustCompile(input string, ops ...expr.Option) *vm.Program {
	if program, err := expr.Compile(input, ops...); err != nil {
		panic(err)
	} else {
		return program
	}
}

func mustRun(program *vm.Program, env any) any {
	if res, err := expr.Run(program, env); err != nil {
		panic(err)
	} else {
		return res
	}
}

type expandInArray struct{}

func (*expandInArray) Enter(*ast.Node) {}

func (*expandInArray) Exit(node *ast.Node) {
	if bi := asBinaryIn(*node); bi != nil {
		if an, ok := bi.Right.(*ast.ArrayNode); ok {
			if len(an.Nodes) == 0 {
				ast.Patch(node, &ast.BoolNode{})
			} else {
				for i, n := range an.Nodes {
					if i == 0 {
						ast.Patch(node, &ast.BinaryNode{
							Operator: "==",
							Left:     bi.Left,
							Right:    n,
						})
					} else {
						ast.Patch(node, &ast.BinaryNode{
							Operator: "or",
							Left:     *node,
							Right: &ast.BinaryNode{
								Operator: "==",
								Left:     bi.Left,
								Right:    n,
							},
						})
					}
				}
			}
			if bi.Operator == "not in" {
				ast.Patch(node, &ast.UnaryNode{
					Operator: "not",
					Node:     *node,
				})
			}
		}
	}
}

type expandInRange struct{}

func (*expandInRange) Enter(*ast.Node) {}

func (*expandInRange) Exit(node *ast.Node) {
	if bi := asBinaryIn(*node); bi != nil {
		if br, ok := bi.Right.(*ast.BinaryNode); ok && br.Operator == ".." {
			if getValue(br.Left) == getValue(br.Right) {
				ast.Patch(node, &ast.BinaryNode{
					Operator: "==",
					Left:     bi.Left,
					Right:    br.Left,
				})
			} else {
				ast.Patch(node, &ast.BinaryNode{
					Operator: "and",
					Left: &ast.BinaryNode{
						Operator: ">=",
						Left:     bi.Left,
						Right:    br.Left,
					},
					Right: &ast.BinaryNode{
						Operator: "<=",
						Left:     bi.Left,
						Right:    br.Right,
					},
				})
			}

			if bi.Operator == "not in" {
				ast.Patch(node, &ast.UnaryNode{
					Operator: "not",
					Node:     *node,
				})
			}
		}
	}
}

func getValue(n ast.Node) any {
	switch a := n.(type) {
	case *ast.NilNode:
		return nil
	case *ast.IntegerNode:
		return a.Value
	case *ast.FloatNode:
		return a.Value
	case *ast.BoolNode:
		return a.Value
	case *ast.StringNode:
		return a.Value
	case *ast.ConstantNode:
		return a.Value
	default:
		return n
	}
}

type distributeAndFoldNot struct{}

func (d *distributeAndFoldNot) Enter(node *ast.Node) {
	if un := asUnaryNot(*node); un != nil {
		if bn, ok := un.Node.(*ast.BinaryNode); ok {
			if op, ok := operatorPairs[bn.Operator]; ok {
				switch bn.Operator {
				case "and", "&&", "or", "||":
					bn.Left = &ast.UnaryNode{
						Operator: un.Operator,
						Node:     bn.Left,
					}
					bn.Right = &ast.UnaryNode{
						Operator: un.Operator,
						Node:     bn.Right,
					}
				}
				bn.Operator = op
				ast.Patch(node, bn)
			}
		} else if n := asUnaryNot(un.Node); n != nil {
			ast.Walk(&n.Node, d)
			ast.Patch(node, n.Node)
		} else if b, ok := un.Node.(*ast.BoolNode); ok {
			b.Value = !b.Value
			ast.Patch(node, b)
		}
	}
}

func (*distributeAndFoldNot) Exit(*ast.Node) {}

func toDNF(node *ast.Node) {
	for limit := 1000; limit >= 0; limit-- {
		f := &dnf{}
		ast.Walk(node, f)
		if !f.applied {
			return
		}
	}
}

type dnf struct {
	depth   int
	applied bool
}

func (f *dnf) Enter(node *ast.Node) {
	if f.depth > 0 {
		f.depth++
	} else if bn, ok := (*node).(*ast.BinaryNode); !ok || (bn.Operator != "and" && bn.Operator != "&&" && bn.Operator != "or" && bn.Operator != "||") {
		f.depth++
	}
}

func (f *dnf) Exit(node *ast.Node) {
	if f.depth > 0 {
		f.depth--
		return
	}

	if ba := asBinaryAnd(*node); ba != nil {
		if bo := asBinaryOr(ba.Left); bo != nil {
			ast.Patch(node, &ast.BinaryNode{
				Operator: bo.Operator,
				Left: &ast.BinaryNode{
					Operator: ba.Operator,
					Left:     bo.Left,
					Right:    ba.Right,
				},
				Right: &ast.BinaryNode{
					Operator: ba.Operator,
					Left:     bo.Right,
					Right:    ba.Right,
				},
			})
			f.applied = true
			return
		}

		if bo := asBinaryOr(ba.Right); bo != nil {
			ast.Patch(node, &ast.BinaryNode{
				Operator: bo.Operator,
				Left: &ast.BinaryNode{
					Operator: ba.Operator,
					Left:     ba.Left,
					Right:    bo.Left,
				},
				Right: &ast.BinaryNode{
					Operator: ba.Operator,
					Left:     ba.Left,
					Right:    bo.Right,
				},
			})
			f.applied = true
			return
		}
	}
}

func asBinaryIn(node ast.Node) *ast.BinaryNode {
	if bn, ok := node.(*ast.BinaryNode); ok && (bn.Operator == "in" || bn.Operator == "not in") {
		return bn
	}
	return nil
}

func asBinaryAnd(node ast.Node) *ast.BinaryNode {
	if bn, ok := node.(*ast.BinaryNode); ok && (bn.Operator == "and" || bn.Operator == "&&") {
		return bn
	}
	return nil
}

func asBinaryOr(node ast.Node) *ast.BinaryNode {
	if bn, ok := node.(*ast.BinaryNode); ok && (bn.Operator == "or" || bn.Operator == "||") {
		return bn
	}
	return nil
}

func asUnaryNot(node ast.Node) *ast.UnaryNode {
	if un, ok := node.(*ast.UnaryNode); ok && (un.Operator == "not" || un.Operator == "!") {
		return un
	}
	return nil
}
