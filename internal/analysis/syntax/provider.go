// Package syntax defines the internal source-syntax boundary used by
// language analyzers.
//
// The boundary deliberately describes a concrete syntax tree without exposing
// a parser implementation. Tree-sitter-backed providers implement it for each
// supported language while analyzers consume one stable syntax contract.
package syntax

import "context"

// Provider parses one source file into a backend-neutral syntax tree.
//
// Providers are associated with a language by their caller; the interface does
// not carry a language switch so adding a parser backend does not require
// modifying consumers. The returned tree is owned by the caller and must be
// closed after the consumer has finished walking it.
type Provider interface {
	Parse(context.Context, Source) (ParseResult, error)
}

// Source is the immutable input supplied to a Provider. A provider may retain
// the content for the lifetime of the returned Tree, so callers must not
// mutate Content until that tree has been closed.
type Source struct {
	Path    string
	Content []byte
}

// ParseResult contains a syntax tree and backend-neutral parse issues. A
// provider may return a usable partial tree together with Issues when the
// parser can recover from malformed source.
type ParseResult struct {
	Tree   Tree
	Issues []Issue
}

// Close releases resources held by the result's tree. It is safe to call on a
// result with no tree.
func (result ParseResult) Close() {
	if result.Tree != nil {
		result.Tree.Close()
	}
}

// Issue describes a parser-reported problem without imposing analyzer-level
// diagnostic codes or severity policy.
type Issue struct {
	Message string
	Range   Range
}

// Tree is an owned syntax tree returned by a Provider.
type Tree interface {
	Root() Node
	Close()
}

// Node is the minimal syntax-tree surface needed by analyzers. Child access
// includes unnamed nodes because comments and punctuation may be represented
// as extras or anonymous grammar nodes by a backend.
type Node interface {
	Type() string
	IsNamed() bool
	IsError() bool
	IsMissing() bool
	Range() Range
	Text() string

	ChildCount() int
	Child(index int) Node
	NamedChildCount() int
	NamedChild(index int) Node
	ChildByFieldName(field string) Node
}

// Point is zero-based and follows the coordinate convention used by
// Tree-sitter: Row is a line number and Column is a byte offset within that
// line.
type Point struct {
	Row    uint32
	Column uint32
}

// Range identifies both the byte span and the row/column span of a syntax
// node. End positions are exclusive.
type Range struct {
	Start     Point
	End       Point
	StartByte uint32
	EndByte   uint32
}

// Walk visits root and its descendants in source order. Returning false from
// visit prunes that node's descendants. A nil root or visitor is ignored.
func Walk(root Node, visit func(Node) bool) {
	if root == nil || visit == nil || !visit(root) {
		return
	}
	for index := 0; index < root.ChildCount(); index++ {
		Walk(root.Child(index), visit)
	}
}
