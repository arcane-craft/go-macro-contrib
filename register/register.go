package register

import (
	"github.com/arcane-craft/go-macro-contrib/inline"
	"github.com/arcane-craft/go-macro-contrib/try"
	"github.com/arcane-craft/go-macro/macro/expandtool"
)

func init() {
	expandtool.Register("github.com/arcane-craft/go-macro-contrib/inline", inline.InlineExpand)
	expandtool.Register("github.com/arcane-craft/go-macro-contrib/try", try.TryExpand)
}
