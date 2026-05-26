package try

// Try0 is a macro stub for (error) callees. Do not call at runtime.
func Try0(err error) {
	panic("Try0 is a macro stub and must not be called at runtime")
}

// Try is a macro stub for (T, error) callees. Do not call at runtime.
func Try[T any](v T, err error) T {
	panic("Try is a macro stub and must not be called at runtime")
}

// Try2 is a macro stub for (A, B, error) callees. Do not call at runtime.
func Try2[A, B any](a A, b B, err error) (A, B) {
	panic("Try2 is a macro stub and must not be called at runtime")
}

// Try3 is a macro stub for (A, B, C, error) callees. Do not call at runtime.
func Try3[A, B, C any](a A, b B, c C, err error) (A, B, C) {
	panic("Try3 is a macro stub and must not be called at runtime")
}
