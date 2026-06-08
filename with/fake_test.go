package with

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sync/atomic"

	"github.com/arcane-craft/go-macro/macro"
)

// fakeCallContext implements macro.CallContext for internal error-path tests.
type fakeCallContext struct {
	fset      *token.FileSet
	file      *ast.File
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

func (c *fakeCallContext) FileSet() *token.FileSet { return c.fset }
func (c *fakeCallContext) File() *ast.File          { return c.file }
func (c *fakeCallContext) LegalSpliceTargets() []macro.SpliceTarget {
	return macro.LegalSpliceTargetsForCall(c.file, c.call)
}
func (c *fakeCallContext) Types() *types.Info            { return c.info }
func (c *fakeCallContext) Package() *types.Package       { return c.pkg }
func (c *fakeCallContext) Call() *ast.CallExpr           { return c.call }
func (c *fakeCallContext) StubName() string              { return c.stub }
func (c *fakeCallContext) SyntaxID() string              { return c.syntaxID }
func (c *fakeCallContext) Site() macro.CallSiteKind      { return c.site }
func (c *fakeCallContext) EnclosingFunc() ast.Node       { return c.enclosing }
func (c *fakeCallContext) MacroPos() token.Pos           { return c.pos }
func (c *fakeCallContext) TempIdent(prefix string) *ast.Ident {
	n := c.counter.Add(1)
	return ast.NewIdent(fmt.Sprintf("%s%d", prefix, n))
}
