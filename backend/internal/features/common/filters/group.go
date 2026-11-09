package filters

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Group struct {
	Logic    string
	Children []Node
}
type Node struct {
	CriterionIndex *int
	Group          *Group
}
type groupParser struct {
	pattern         string
	tokens          []groupToken
	position, count int
	used            map[int]bool
}
type groupToken struct {
	kind, text string
	index      int
}

func nodeIndex(i int) Node {
	return Node{
		CriterionIndex: &i,
	}
}
func andAll(count int) *Group {
	g := &Group{
		Logic: "AND",
	}
	for i := 0; i < count; i++ {
		g.Children = append(g.Children, nodeIndex(i))
	}
	return g
}
func BuildGroup(pattern string, count int) (*Group, error) {
	if count <= 0 {
		return nil, nil
	}
	if strings.TrimSpace(pattern) == "" {
		return andAll(count), nil
	}
	p := groupParser{
		pattern: pattern,
		count:   count,
		used:    map[int]bool{},
	}
	if err := p.tokenize(); err != nil {
		return nil, err
	}
	root, err := p.expr(1)
	if err != nil {
		return nil, err
	}
	if p.position != len(p.tokens) {
		return nil, invalid("Invalid group pattern.", nil)
	}
	var g *Group
	if root.Group != nil {
		g = root.Group
	} else {
		g = &Group{
			Logic: "AND",
			Children: []Node{
				root,
			},
		}
	}
	missing := []Node{}
	for i := 0; i < count; i++ {
		if !p.used[i] {
			missing = append(missing, nodeIndex(i))
		}
	}
	if len(missing) > 0 {
		g = &Group{
			Logic: "AND",
			Children: append([]Node{
				{
					Group: g,
				},
			}, missing...),
		}
	}
	return g, nil
}
func (p *groupParser) tokenize() error {
	s := p.pattern
	for i := 0; i < len(s); {
		if unicode.IsSpace(rune(s[i])) {
			i++
			continue
		}
		if s[i] == '(' || s[i] == ')' {
			p.tokens = append(p.tokens, groupToken{
				kind: string(s[i]),
				text: string(s[i]),
			})
			i++
			continue
		}
		if s[i] == '{' {
			end := strings.IndexByte(s[i+1:], '}')
			if end < 0 {
				return invalid("Group pattern contains an unclosed criterion index.", nil)
			}
			end += i + 1
			raw := strings.TrimSpace(s[i+1 : end])
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 {
				return invalid(fmt.Sprintf("Invalid criterion index '{%s}' in group pattern.", raw), nil)
			}
			p.tokens = append(p.tokens, groupToken{
				kind:  "index",
				index: n,
				text:  s[i : end+1],
			})
			i = end + 1
			continue
		}
		matched := false
		for _, k := range []string{
			"AND",
			"OR",
			"NOT",
		} {
			if len(s)-i >= len(k) && strings.EqualFold(s[i:i+len(k)], k) && (len(s) == i+len(k) || !unicode.IsLetter(rune(s[i+len(k)])) && !unicode.IsDigit(rune(s[i+len(k)]))) {
				p.tokens = append(p.tokens, groupToken{
					kind: k,
					text: s[i : i+len(k)],
				})
				i += len(k)
				matched = true
				break
			}
		}
		if !matched {
			return invalid(fmt.Sprintf("Unsupported token near '%s' in group pattern.", s[i:]), nil)
		}
	}
	return nil
}
func (p *groupParser) expr(min int) (Node, error) {
	left, err := p.operand()
	if err != nil {
		return Node{}, err
	}
	for p.position < len(p.tokens) {
		t := p.tokens[p.position]
		prec := 0
		if t.kind == "OR" {
			prec = 1
		}
		if t.kind == "AND" {
			prec = 2
		}
		if prec < min {
			break
		}
		p.position++
		right, err := p.expr(prec + 1)
		if err != nil {
			return Node{}, err
		}
		left = Node{
			Group: &Group{
				Logic: t.kind,
				Children: []Node{
					left,
					right,
				},
			},
		}
	}
	return left, nil
}
func (p *groupParser) operand() (Node, error) {
	if p.position >= len(p.tokens) {
		return Node{}, invalid("Group pattern cannot end with an operator.", nil)
	}
	t := p.tokens[p.position]
	p.position++
	switch t.kind {
	case "index":
		if t.index >= p.count {
			return Node{}, invalid(fmt.Sprintf("Criterion index '%d' is outside the available range.", t.index), nil)
		}
		p.used[t.index] = true
		return nodeIndex(t.index), nil
	case "(":
		child, err := p.expr(1)
		if err != nil {
			return Node{}, err
		}
		if p.position >= len(p.tokens) || p.tokens[p.position].kind != ")" {
			return Node{}, invalid("Unmatched '(' in group pattern.", nil)
		}
		p.position++
		return child, nil
	case "NOT":
		if p.position < 2 || p.tokens[p.position-2].kind != "(" {
			return Node{}, invalid("NOT must be wrapped in parentheses, for example '(NOT {1})'.", nil)
		}
		child, err := p.operand()
		if err != nil {
			return Node{}, err
		}
		return Node{
			Group: &Group{
				Logic: "NOT",
				Children: []Node{
					child,
				},
			},
		}, nil

	default:
		return Node{}, invalid("Invalid group pattern.", nil)
	}
}
