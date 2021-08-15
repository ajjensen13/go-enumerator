# go-enumerator
go-enumerator is a code generation tool that
generates methods for constant types that
allow them to behave similar to enums in other languages.

go-enumerator is designed to be invoked by `go generate`, 
but it can be used as a command-line tool as well.

See [https://pkg.go.dev/github.com/ajjensen13/go-enumerator](https://pkg.go.dev/github.com/ajjensen13/go-enumerator)
for additional documentation.

## Installation
Installation is easy, just install the package using the `go install` tool.

```shell
go install github.com/ajjensen13/go-enumerator
```

## Overview
Below is an example of the intended use for go-enumerate.
When invoked via `go generate` all command line arguments are optional.
The tool will use the GOFILE, GOPACKAGE, and GOLINE environment variables
to find the type declaration immediately following to `go:generate` comment.

```go
//go:generate go-enumerator
type Kind int

const (
	Kind1
	Kind2
)
```

In this case, we found the `Kind` type, which is a suitable type for generating an enum definition for. 
The following methods are created in a new file with the default file name.

```go
// String implements fmt.Stringer
func (k Kind) String() string { /* omitted for brevity */ }

// Scan implements fmt.Scanner
func (k *Kind) Scan(ss fmt.ScanState, verb rune) error { /* omitted for brevity */ }

// Defined returns true if k holds a defined value
func (k Kind) Defined() bool { /* omitted for brevity */ }

// Next returns the next defined value after k
func (k Kind) Next() Kind { /* omitted for brevity */ }

```

`String()` and `Scan()` can be used in conjunction with the `fmt` package to parse
and encode values into human-friendly representations.

`Next()` can be used to loop through all defined values for an _enum_.

## Remarks
* go-enumerator was inspired by [stringer](https://pkg.go.dev/golang.org/x/tools/cmd/stringer)
* Examples for how to use the generated code can be found at [https://pkg.go.dev/github.com/ajjensen13/go-enumerator/example](https://pkg.go.dev/github.com/ajjensen13/go-enumerator/example])