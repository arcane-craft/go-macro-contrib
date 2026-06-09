package inline

//macro: inline
// Inline0 is a macro stub for void callees (pass func() { ... }). Do not call at runtime.
func Inline0(f func()) {
	panic("Inline0 is a macro stub and must not be called at runtime")
}

//macro: inline
// Inline is a macro stub for single-value callees. Do not call at runtime.
func Inline[T any](v T) T {
	panic("Inline is a macro stub and must not be called at runtime")
}

//macro: inline
// Inline2 is a macro stub for two-value callees. Do not call at runtime.
func Inline2[A, B any](a A, b B) (A, B) {
	panic("Inline2 is a macro stub and must not be called at runtime")
}

//macro: inline
// Inline3 is a macro stub for three-value callees. Do not call at runtime.
func Inline3[A, B, C any](a A, b B, c C) (A, B, C) {
	panic("Inline3 is a macro stub and must not be called at runtime")
}
