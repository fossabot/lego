package config

import (
	"io"

	toml "github.com/pelletier/go-toml"
	"github.com/pelletier/go-toml/query"
	"github.com/pkg/errors"
)

var q = mustCompile(query.Compile("$..*"))

// Tree is a configuration tree
type Tree interface {
	Keys() []string
	Has(key string) bool
	Get(key string) Tree
	Unmarshal(v interface{}) error
	String() string
}

// LoadTree loads r into a config tree
func LoadTree(r io.Reader) (Tree, error) {
	t, err := toml.LoadReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "error loading config tree")
	}

	// Replace all environment variables with their value
	results := q.Execute(t)
	for _, item := range results.Values() {
		switch v := item.(type) {
		case *toml.Tree:
			for _, key := range v.Keys() {
				v.Set(key, ValuesOf(v.Get(key)))
			}
		case []*toml.Tree:
			for _, tree := range v {
				for _, key := range tree.Keys() {
					tree.Set(key, ValuesOf(tree.Get(key)))
				}
			}
		}
	}

	return &tree{t: t}, nil
}

// tree wraps a TOML tree
type tree struct {
	t *toml.Tree
}

func (t *tree) Keys() []string {
	return t.t.Keys()
}

func (t *tree) Has(key string) bool {
	_, ok := t.t.Get(key).(*toml.Tree)
	return ok
}

func (t *tree) Get(key string) Tree {
	child, ok := t.t.Get(key).(*toml.Tree)
	if !ok {
		return &nullTree{}
	}
	return &tree{t: child}
}

func (t *tree) Unmarshal(v interface{}) error {
	err := t.t.Unmarshal(v)
	if err != nil {
		return errors.Wrap(err, "cannot unmarshal config tree")
	}
	return nil
}

func (t *tree) String() string {
	s, _ := t.t.ToTomlString()
	return s
}

// nullTree is a tree that does not do anything (null pattern)
type nullTree struct{}

func (t *nullTree) Keys() []string                { return nil }
func (t *nullTree) Has(key string) bool           { return false }
func (t *nullTree) Get(key string) Tree           { return t }
func (t *nullTree) Unmarshal(v interface{}) error { return nil }
func (t *nullTree) String() string                { return "" }

func mustCompile(q *query.Query, err error) *query.Query {
	if err != nil {
		panic(err)
	}
	return q
}
