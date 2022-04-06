package table_header

import (
	"fmt"
	"sort"

	"github.com/russross/blackfriday/v2/table_header/parser"
	"github.com/shopspring/decimal"
)

type Node struct {
	Value    string
	Children []*Node
	Tags     map[string]interface{}
}

func (root *Node) TableHeader() (mt TableHeader) {
	var (
		sort = func(this []*Header) {
			var compare = func(x, y int) int {
				if x < y {
					return -1
				}
				if x == y {
					return 0
				}
				return 1
			}
			sort.Slice(this, func(i, j int) bool {
				var (
					a, b = this[i], this[j]
					pri  = compare(a.Row, b.Row)
					sec  = compare(a.Col, b.Col)
				)
				if pri != 0 {
					return pri < 0
				}
				return sec < 0
			})
		}
		lcm = func(a, b int) int {
			c := a * b
			for b > 0 {
				t := b
				b = a % b
				a = t
			}
			return c / a
		}
		rowsToUse, width func(t *Node) int
		getCells         func(depth int, t *Node, row, col, rowsLeft int) []*Header
	)
	rowsToUse = func(t *Node) int {
		var childrenRows int
		if len(t.Children) > 0 {
			childrenRows++
		}

		for _, child := range t.Children {
			childrenRows = lcm(childrenRows, rowsToUse(child))
		}
		return 1 + childrenRows
	}
	width = func(t *Node) int {
		if len(t.Children) == 0 {
			return 1
		}
		w := 0
		for _, child := range t.Children {
			w += width(child)
		}
		return w
	}
	getCells = func(depth int, t *Node, row, col, rowsLeft int) (cells []*Header) {
		// Add top-most cell corresponding to the root of the current tree.
		rootRows := rowsLeft / rowsToUse(t)
		th := &Header{t.Value, t.Tags, len(t.Children) == 0, depth, row, col, rootRows, width(t)}
		cells = append(cells, th)
		for _, child := range t.Children {
			cells = append(cells, getCells(depth+1, child, row+rootRows, col, rowsLeft-rootRows)...)
			col += width(child)
		}
		if (row + 1) > mt.NumRows {
			mt.NumRows = row + 1
		}
		if th.IsLeaf {
			mt.Leafs = append(mt.Leafs, th)
		}
		return
	}
	cells := getCells(0, root, 0, 0, rowsToUse(root))
	sort(cells)
	mt.Rows = make([][]*Header, mt.NumRows)

	for _, cell := range cells {
		mt.Rows[cell.Row] = append(mt.Rows[cell.Row], cell)
	}
	mt.Rows = mt.Rows[1:]
	mt.NumRows--

	return
}

type Header struct {
	Value  string
	Tags   map[string]interface{}
	IsLeaf bool
	Depth,
	Row,
	Col,
	RowSpan,
	ColSpan int
}

func (m Header) Tag() string {
	return fmt.Sprintf("%s [%d %d %d]", m.Value, m.Row, m.RowSpan, m.ColSpan)
}

type TableHeader struct {
	Rows    [][]*Header
	NumRows int
	Leafs   []*Header
}

func Parse(input string) (_ *Node, err error) {
	fs := parser.NewFileSet()
	p := parser.NewParser(fs.AddFile("main", -1, len(input)), []byte(input), nil)
	var actual *parser.File
	if actual, err = p.ParseFile(); err != nil {
		return
	}
	for _, s := range actual.Stmts {
		switch t := s.(type) {
		case *parser.ExprStmt:
			if vl, ok := t.Expr.(*parser.ValuesLit); ok {
				return BuildNode(vl)
			}
		}
	}
	return
}

func BuildNode(vl *parser.ValuesLit) (th *Node, err error) {
	th = &Node{}
	var tags map[string]interface{}
	for _, el := range vl.Elements {
		if el.Tags == nil {
			tags = nil
		} else {
			if tags, err = BuildMap(el.Tags); err != nil {
				return nil, err
			}
		}
		if el.Value == nil {
			th.Children = append(th.Children, &Node{Value: el.Key, Tags: tags})
		} else {
			switch t := el.Value.(type) {
			case *parser.StringLit:
				th.Children = append(th.Children, &Node{Value: t.Value, Tags: tags})
			case *parser.ValuesLit:
				var child *Node
				if child, err = BuildNode(t); err != nil || child == nil {
					return
				}
				child.Value = el.Key
				child.Tags = tags
				th.Children = append(th.Children, child)
			case *parser.Ident:
				th.Children = append(th.Children, &Node{Value: el.Key, Children: []*Node{{Value: t.Name}}, Tags: tags})
			default:
				err = fmt.Errorf("invalid value type %T", el.Value)
				return
			}
		}
	}
	return
}

func BuildMap(vl *parser.ValuesLit) (th map[string]interface{}, err error) {
	th = make(map[string]interface{})
	for _, el := range vl.Elements {
		if el.Value == nil {
			th[el.Key] = true
		} else {
			switch t := el.Value.(type) {
			case *parser.StringLit:
				th[el.Key] = t.Value
			case *parser.ValuesLit:
				var child map[string]interface{}
				if child, err = BuildMap(t); err != nil || child == nil {
					return
				}
				th[el.Key] = child
			case *parser.Ident:
				th[el.Key] = t.Name
			case *parser.Number:
				th[el.Key], err = decimal.NewFromString(t.Value)
				if err != nil {
					return
				}
			default:
				err = fmt.Errorf("invalid value type %T", el.Value)
				return
			}
		}
	}
	return
}
