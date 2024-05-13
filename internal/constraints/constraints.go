// Package constraints should only be used as a blank import. When
// imported it will cause `go build` to fail with an obvious and clean
// message if the constraints defined in the package are not met.
package constraints

var (
	// Only available when on Go v1.13 or more.
	_ = Go120

	// Only available on 64-bit architectures.
	_ = GoArch64
)
