// Package messagefilter is a small smart-filter expression engine for Kafka
// messages (task MSG-24). It compiles a predicate expression into a Filter that
// can be evaluated against api.Message values.
//
// Grammar (recursive descent, no external deps):
//
//	expr       := orExpr
//	orExpr     := andExpr ( ("OR"|"or"|"||") andExpr )*
//	andExpr    := comparison ( ("AND"|"and"|"&&") comparison )*
//	comparison := "(" expr ")" | field OP operand
//	field      := key | value | partition | offset | header.NAME | headers.NAME
//	OP         := contains | == | != | > | < | >= | <=
//	operand    := "quoted" | 'quoted' | number | bareword
package messagefilter

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
)

// Filter is a compiled smart-filter expression.
type Filter struct {
	expr string
	root node
}

// Compile parses expr into a Filter. Empty/whitespace expr => error.
func Compile(expr string) (*Filter, error) {
	if strings.TrimSpace(expr) == "" {
		return nil, fmt.Errorf("empty expression")
	}
	toks, err := tokenize(expr)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	root, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if !p.done() {
		return nil, fmt.Errorf("unexpected token %q", p.peek().val)
	}
	return &Filter{expr: expr, root: root}, nil
}

// Eval evaluates the compiled filter against a message. err is non-nil only on
// evaluation errors (e.g. a numeric comparison against a non-numeric field
// value).
func (f *Filter) Eval(m api.Message) (bool, error) {
	return f.root.eval(m)
}

// Expr returns the original expression string.
func (f *Filter) Expr() string { return f.expr }

// ID returns a deterministic short id derived from a hash of the expression
// (same expression => same id).
func (f *Filter) ID() string {
	sum := sha256.Sum256([]byte(f.expr))
	return hex.EncodeToString(sum[:])[:8]
}

// Test compiles expr and evaluates it against each sample, returning per-sample
// results. compileErr is set if compilation failed (results nil then). evalErr
// is set if any sample fails to evaluate.
func Test(expr string, samples []api.Message) (results []bool, compileErr error, evalErr error) {
	f, err := Compile(expr)
	if err != nil {
		return nil, err, nil
	}
	results = make([]bool, len(samples))
	for i, m := range samples {
		ok, err := f.Eval(m)
		if err != nil {
			return results, nil, err
		}
		results[i] = ok
	}
	return results, nil, nil
}

// --- AST ---

type node interface {
	eval(m api.Message) (bool, error)
}

type orNode struct{ left, right node }

func (n orNode) eval(m api.Message) (bool, error) {
	l, err := n.left.eval(m)
	if err != nil {
		return false, err
	}
	if l {
		return true, nil
	}
	return n.right.eval(m)
}

type andNode struct{ left, right node }

func (n andNode) eval(m api.Message) (bool, error) {
	l, err := n.left.eval(m)
	if err != nil {
		return false, err
	}
	if !l {
		return false, nil
	}
	return n.right.eval(m)
}

type fieldKind int

const (
	fKey fieldKind = iota
	fValue
	fPartition
	fOffset
	fHeader
)

type cmpNode struct {
	kind    fieldKind
	header  string // only for fHeader
	op      string
	operand string
}

func (n cmpNode) numeric() bool { return n.kind == fPartition || n.kind == fOffset }

func (n cmpNode) strVal(m api.Message) string {
	switch n.kind {
	case fKey:
		return m.Key
	case fValue:
		return m.Value
	case fPartition:
		return strconv.FormatInt(int64(m.Partition), 10)
	case fOffset:
		return strconv.FormatInt(m.Offset, 10)
	case fHeader:
		for _, h := range m.Headers {
			if h.Key == n.header {
				return h.Value
			}
		}
		return ""
	}
	return ""
}

func (n cmpNode) intVal(m api.Message) int64 {
	if n.kind == fPartition {
		return int64(m.Partition)
	}
	return m.Offset
}

func (n cmpNode) eval(m api.Message) (bool, error) {
	// contains: case-insensitive substring match against the string form.
	if n.op == "contains" {
		return strings.Contains(
			strings.ToLower(n.strVal(m)),
			strings.ToLower(n.operand),
		), nil
	}

	if n.numeric() {
		return n.evalNumeric(m)
	}
	return n.evalString(m)
}

func (n cmpNode) evalNumeric(m api.Message) (bool, error) {
	rhs, err := strconv.ParseInt(n.operand, 10, 64)
	if err != nil {
		return false, fmt.Errorf("operand %q is not numeric for field comparison", n.operand)
	}
	lhs := n.intVal(m)
	switch n.op {
	case "==":
		return lhs == rhs, nil
	case "!=":
		return lhs != rhs, nil
	case ">":
		return lhs > rhs, nil
	case "<":
		return lhs < rhs, nil
	case ">=":
		return lhs >= rhs, nil
	case "<=":
		return lhs <= rhs, nil
	}
	return false, fmt.Errorf("unknown operator %q", n.op)
}

func (n cmpNode) evalString(m api.Message) (bool, error) {
	lhs := n.strVal(m)
	rhs := n.operand
	switch n.op {
	case "==":
		return lhs == rhs, nil
	case "!=":
		return lhs != rhs, nil
	case ">":
		return lhs > rhs, nil
	case "<":
		return lhs < rhs, nil
	case ">=":
		return lhs >= rhs, nil
	case "<=":
		return lhs <= rhs, nil
	}
	return false, fmt.Errorf("unknown operator %q", n.op)
}

// --- tokenizer ---

type tokKind int

const (
	kWord tokKind = iota
	kString
	kOp
	kLogic
	kLParen
	kRParen
)

type token struct {
	kind tokKind
	val  string
}

func isOpStart(c byte) bool {
	switch c {
	case '=', '!', '<', '>', '&', '|':
		return true
	}
	return false
}

func tokenize(s string) ([]token, error) {
	var toks []token
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '(':
			toks = append(toks, token{kLParen, "("})
			i++
		case c == ')':
			toks = append(toks, token{kRParen, ")"})
			i++
		case c == '"' || c == '\'':
			quote := c
			i++
			start := i
			for i < len(s) && s[i] != quote {
				i++
			}
			if i >= len(s) {
				return nil, fmt.Errorf("unterminated string literal")
			}
			toks = append(toks, token{kString, s[start:i]})
			i++ // closing quote
		case c == '&' && i+1 < len(s) && s[i+1] == '&':
			toks = append(toks, token{kLogic, "&&"})
			i += 2
		case c == '|' && i+1 < len(s) && s[i+1] == '|':
			toks = append(toks, token{kLogic, "||"})
			i += 2
		case c == '=' || c == '!':
			if i+1 < len(s) && s[i+1] == '=' {
				toks = append(toks, token{kOp, s[i : i+2]})
				i += 2
			} else {
				return nil, fmt.Errorf("unexpected token %q", string(c))
			}
		case c == '<' || c == '>':
			if i+1 < len(s) && s[i+1] == '=' {
				toks = append(toks, token{kOp, s[i : i+2]})
				i += 2
			} else {
				toks = append(toks, token{kOp, string(c)})
				i++
			}
		default:
			// bareword: run until whitespace, paren, quote, or operator start
			start := i
			for i < len(s) {
				b := s[i]
				if b == ' ' || b == '\t' || b == '\n' || b == '\r' ||
					b == '(' || b == ')' || b == '"' || b == '\'' || isOpStart(b) {
					break
				}
				i++
			}
			toks = append(toks, token{kWord, s[start:i]})
		}
	}
	return toks, nil
}

// --- parser ---

type parser struct {
	toks []token
	pos  int
}

func (p *parser) done() bool { return p.pos >= len(p.toks) }

func (p *parser) peek() token {
	if p.done() {
		return token{kWord, ""}
	}
	return p.toks[p.pos]
}

func (p *parser) next() token {
	t := p.peek()
	p.pos++
	return t
}

func isOrTok(t token) bool {
	return (t.kind == kLogic && t.val == "||") ||
		(t.kind == kWord && (t.val == "OR" || t.val == "or"))
}

func isAndTok(t token) bool {
	return (t.kind == kLogic && t.val == "&&") ||
		(t.kind == kWord && (t.val == "AND" || t.val == "and"))
}

func (p *parser) parseExpr() (node, error) {
	return p.parseOr()
}

func (p *parser) parseOr() (node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for !p.done() && isOrTok(p.peek()) {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = orNode{left, right}
	}
	return left, nil
}

func (p *parser) parseAnd() (node, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for !p.done() && isAndTok(p.peek()) {
		p.next()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = andNode{left, right}
	}
	return left, nil
}

func (p *parser) parseComparison() (node, error) {
	if p.done() {
		return nil, fmt.Errorf("unexpected end of expression")
	}
	if p.peek().kind == kLParen {
		p.next()
		inner, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.done() || p.peek().kind != kRParen {
			return nil, fmt.Errorf("unbalanced parentheses")
		}
		p.next() // consume ')'
		return inner, nil
	}

	// field OP operand
	ft := p.next()
	if ft.kind != kWord {
		return nil, fmt.Errorf("unexpected token %q, expected field", ft.val)
	}
	leaf, err := parseField(ft.val)
	if err != nil {
		return nil, err
	}

	if p.done() {
		return nil, fmt.Errorf("expected operator after field %q", ft.val)
	}
	ot := p.next()
	op, err := parseOp(ot)
	if err != nil {
		return nil, err
	}
	leaf.op = op

	if p.done() {
		return nil, fmt.Errorf("expected operand after operator %q", op)
	}
	vt := p.next()
	if vt.kind != kString && vt.kind != kWord {
		return nil, fmt.Errorf("unexpected token %q, expected operand", vt.val)
	}
	leaf.operand = vt.val
	return leaf, nil
}

func parseField(name string) (*cmpNode, error) {
	switch name {
	case "key":
		return &cmpNode{kind: fKey}, nil
	case "value":
		return &cmpNode{kind: fValue}, nil
	case "partition":
		return &cmpNode{kind: fPartition}, nil
	case "offset":
		return &cmpNode{kind: fOffset}, nil
	}
	if h, ok := strings.CutPrefix(name, "header."); ok && h != "" {
		return &cmpNode{kind: fHeader, header: h}, nil
	}
	if h, ok := strings.CutPrefix(name, "headers."); ok && h != "" {
		return &cmpNode{kind: fHeader, header: h}, nil
	}
	return nil, fmt.Errorf("unknown field %q", name)
}

func parseOp(t token) (string, error) {
	if t.kind == kOp {
		return t.val, nil
	}
	if t.kind == kWord && strings.ToLower(t.val) == "contains" {
		return "contains", nil
	}
	return "", fmt.Errorf("unknown operator %q", t.val)
}
