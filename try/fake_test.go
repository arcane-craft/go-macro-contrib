package try

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sync/atomic"

	"github.com/arcane-craft/go-macro/macro"
)

// fakeContext implements macro.Context for internal error-path tests.
type fakeContext struct {
	fset      *token.FileSet
	info      *types.Info
	pkg       *types.Package
	call      *ast.CallExpr
	stub      string
	syntaxID  string
	site      macro.CallSiteKind
	enclosing ast.Node
	pos       token.Pos
	counter   atomic.Uint64
}

func (c *fakeContext) FileSet() *token.FileSet       { return c.fset }
func (c *fakeContext) Types() *types.Info            { return c.info }
func (c *fakeContext) Package() *types.Package       { return c.pkg }
func (c *fakeContext) Call() *ast.CallExpr           { return c.call }
func (c *fakeContext) StubName() string              { return c.stub }
func (c *fakeContext) SyntaxID() string              { return c.syntaxID }
func (c *fakeContext) Site() macro.CallSiteKind      { return c.site }
func (c *fakeContext) EnclosingFunc() ast.Node       { return c.enclosing }
func (c *fakeContext) MacroPos() token.Pos           { return c.pos }
func (c *fakeContext) TempIdent(prefix string) *ast.Ident {
	n := c.counter.Add(1)
	return ast.NewIdent(fmt.Sprintf("%s%d", prefix, n))
}
