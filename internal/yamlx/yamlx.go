// Package yamlx edits YAML documents as a tree of nodes, so the commands
// that rewrite a card change only the keys they own and keep everything
// else, comments and the order of the keys included.
package yamlx

import (
	"bytes"
	"fmt"
	"slices"

	"go.yaml.in/yaml/v3"
)

// Parse reads data as one YAML document whose top is a mapping, and returns
// the document node.
func Parse(data []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("the card is not a YAML mapping")
	}
	return &doc, nil
}

// Encode writes doc back with the two-space indent the cards use.
func Encode(doc *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Field returns the value of key in a mapping node, or nil.
func Field(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// Set sets key in a mapping node. A key that is not there yet goes right
// after the first of after that is, so a card keeps reading in the same
// order, and at the end when none of them is.
func Set(m *yaml.Node, key string, value *yaml.Node, after ...string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = value
			return
		}
	}
	pos := len(m.Content)
	for _, a := range after {
		if i := keyIndex(m, a); i >= 0 {
			pos = i + 2
			break
		}
	}
	rest := slices.Clone(m.Content[pos:])
	m.Content = append(append(m.Content[:pos:pos], Scalar(key), value), rest...)
}

// Remove deletes key from a mapping node, if it is there.
func Remove(m *yaml.Node, key string) {
	if i := keyIndex(m, key); i >= 0 {
		m.Content = append(m.Content[:i:i], m.Content[i+2:]...)
	}
}

func keyIndex(m *yaml.Node, key string) int {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// Scalar is a plain string node.
func Scalar(s string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
}

// List is a sequence of strings, written on one line.
func List(values []string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
	for _, v := range values {
		n.Content = append(n.Content, Scalar(v))
	}
	return n
}
