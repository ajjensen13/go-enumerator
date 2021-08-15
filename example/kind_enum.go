// Code generated by "go-enumerator"; DO NOT EDIT.

package example

import "fmt"

// String implements fmt.Stringer. If !k.Defined(), then a generated string is returned based on k's value.
func (k Kind) String() string {
	switch k {
	case Kind1:
		return "Kind1"
	case Kind2:
		return "Kind2"
	}
	return fmt.Sprintf("Kind(%d)", k)
}

// Defined returns true if k holds a defined value.
func (k Kind) Defined() bool {
	switch k {
	case 0, 1:
		return true
	default:
		return false
	}
}

// Scan implements fmt.Scanner. Use fmt.Scan() to parse strings into Kind values
func (k *Kind) Scan(scanState fmt.ScanState, verb rune) error {
	token, err := scanState.Token(true, nil)
	if err != nil {
		return err
	}

	switch string(token) {
	case "Kind1":
		*k = Kind1
	case "Kind2":
		*k = Kind2
	default:
		return fmt.Errorf("unknown Kind value: %s", token)
	}
	return nil
}

// Next returns the next defined Kind. If k is not defined, then Next returns the first defined value.
// The order that defined values are returned is undefined.
// The only guarantee is that all defined values will be returned before Next starts cycling through previous values again.
// The order will be consistent for a given program, but the order may change if the program is re-compiled.
func (k Kind) Next() Kind {
	switch k {
	case Kind1:
		return Kind2
	case Kind2:
		return Kind1
	default:
		return Kind1
	}
}

func _() {
	var x [1]struct{}
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the go-enumerator command to generate them again.
	_ = x[Kind1-0]
	_ = x[Kind2-1]
}
