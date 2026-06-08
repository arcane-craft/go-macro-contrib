package with

import "io"

//macro: syntax-with
// With is a macro stub for (T, error) callees where T is io.Closer. Do not call at runtime.
func With[T io.Closer](v T, err error) T {
	panic("With is a macro stub and must not be called at runtime")
}
