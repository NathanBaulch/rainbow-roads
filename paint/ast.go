package paint

import (
	"fmt"

	"github.com/expr-lang/expr/ast"
)

type Visitor interface {
	Enter(node *ast.Node)
	Exit(node *ast.Node)
}

func Walk(node *ast.Node, v Visitor) {
	if *node == nil {
		return
	}

	v.Enter(node)

	switch n := (*node).(type) {
	case *ast.NilNode:
	case *ast.IdentifierNode:
	case *ast.IntegerNode:
	case *ast.FloatNode:
	case *ast.BoolNode:
	case *ast.StringNode:
	case *ast.ConstantNode:
	case *ast.UnaryNode:
		Walk(&n.Node, v)
	case *ast.BinaryNode:
		Walk(&n.Left, v)
		Walk(&n.Right, v)
	case *ast.ChainNode:
		Walk(&n.Node, v)
	case *ast.MemberNode:
		Walk(&n.Node, v)
		Walk(&n.Property, v)
	case *ast.SliceNode:
		Walk(&n.Node, v)
		if n.From != nil {
			Walk(&n.From, v)
		}
		if n.To != nil {
			Walk(&n.To, v)
		}
	case *ast.CallNode:
		Walk(&n.Callee, v)
		for i := range n.Arguments {
			Walk(&n.Arguments[i], v)
		}
	case *ast.BuiltinNode:
		for i := range n.Arguments {
			Walk(&n.Arguments[i], v)
		}
	case *ast.ClosureNode:
		Walk(&n.Node, v)
	case *ast.PointerNode:
	case *ast.VariableDeclaratorNode:
		Walk(&n.Value, v)
		Walk(&n.Expr, v)
	case *ast.ConditionalNode:
		Walk(&n.Cond, v)
		Walk(&n.Exp1, v)
		Walk(&n.Exp2, v)
	case *ast.ArrayNode:
		for i := range n.Nodes {
			Walk(&n.Nodes[i], v)
		}
	case *ast.MapNode:
		for i := range n.Pairs {
			Walk(&n.Pairs[i], v)
		}
	case *ast.PairNode:
		Walk(&n.Key, v)
		Walk(&n.Value, v)
	default:
		panic(fmt.Sprintf("undefined node type (%T)", node))
	}

	v.Exit(node)
}
