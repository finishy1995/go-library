package mongodb

import (
	"errors"
	"fmt"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/finishy1995/go-library/storage/src/tools"
	"go.mongodb.org/mongo-driver/bson"
	"strings"
)

var (
	keyWordExtended = map[string]string{
		"AND": "and",
		"OR":  "or",
		"NOT": "not",
		"=":   "==",
	}
)

func replaceWithExtended(expr string) string {
	tmp := expr
	for key, value := range keyWordExtended {
		tmp = strings.ReplaceAll(tmp, " "+key+" ", " "+value+" ")
	}
	return tmp
}

func replaceWithValue(expr string, args ...interface{}) string {
	realExpr := strings.ReplaceAll(expr, "?", "%v")

	return fmt.Sprintf(realExpr, args...)
}

func getRootNode(expr string, args ...interface{}) (*ast.BinaryNode, error) {
	realExpr := replaceWithExtended(expr)
	realExpr = replaceWithValue(realExpr, args...)

	// 使用配置解析表达式
	tree, err := parser.Parse(realExpr)
	if err != nil {
		return nil, err
	}

	// 断言根节点是二元节点
	node, ok := tree.Node.(*ast.BinaryNode)
	if !ok {
		return nil, errors.New("root node is not a BinaryNode")
	}
	return node, nil
}

// buildFilterFromAST 根据 AST 节点构建 MongoDB 查询条件
func buildFilterFromAST(node *ast.BinaryNode) (bson.D, error) {
	if node == nil {
		return bson.D{}, fmt.Errorf("node is nil")
	}

	switch node.Operator {
	case "and":
		left, err := buildFilterFromAST(node.Left.(*ast.BinaryNode))
		if err != nil {
			return nil, err
		}
		right, err := buildFilterFromAST(node.Right.(*ast.BinaryNode))
		if err != nil {
			return nil, err
		}
		return bson.D{{"$and", bson.A{left, right}}}, nil

	case "or":
		left, err := buildFilterFromAST(node.Left.(*ast.BinaryNode))
		if err != nil {
			return nil, err
		}
		right, err := buildFilterFromAST(node.Right.(*ast.BinaryNode))
		if err != nil {
			return nil, err
		}
		return bson.D{{"$or", bson.A{left, right}}}, nil

	case "not":
		// TODO: MongoDB 的 `$not` 运算符的行为可能与一般的 not 运算符有所不同
		operand, err := buildFilterFromAST(node.Left.(*ast.BinaryNode))
		if err != nil {
			return nil, err
		}
		return bson.D{{"$not", operand}}, nil

	default:
		// 处理比较运算符
		leftVal, err := getOperandValue(node.Left)
		if err != nil {
			return nil, fmt.Errorf("left operand error: %v", err)
		}
		left, ok := leftVal.(string)
		if !ok {
			return nil, fmt.Errorf("unsupported operand type")
		}
		left = tools.LowerAllChar(left)

		rightVal, err := getOperandValue(node.Right)
		if err != nil {
			return nil, fmt.Errorf("right operand error: %v", err)
		}

		switch node.Operator {
		case ">":
			return bson.D{{left, bson.M{"$gt": rightVal}}}, nil
		case "<":
			return bson.D{{left, bson.M{"$lt": rightVal}}}, nil
		case "==":
			return bson.D{{left, bson.M{"$eq": rightVal}}}, nil
		case ">=":
			return bson.D{{left, bson.M{"$gte": rightVal}}}, nil
		case "<=":
			return bson.D{{left, bson.M{"$lte": rightVal}}}, nil
		default:
			return nil, fmt.Errorf("unsupported operator: %s", node.Operator)
		}
	}
}

func getOperandValue(n ast.Node) (interface{}, error) {
	switch operand := n.(type) {
	case *ast.IdentifierNode:
		return operand.Value, nil
	case *ast.IntegerNode:
		return operand.Value, nil
	case *ast.FloatNode:
		return operand.Value, nil
	case *ast.BoolNode:
		return operand.Value, nil
	case *ast.StringNode:
		return operand.Value, nil
	default:
		return nil, fmt.Errorf("unsupported operand type")
	}
}
