package example

import (
	"fmt"
	"testing"
)

func TestKind_Defined(t *testing.T) {
	tests := []struct {
		name string
		e    Kind
		want bool
	}{
		{
			"Kind1",
			Kind1,
			true,
		},
		{
			"Kind2",
			Kind2,
			true,
		},
		{
			"undefined",
			-1,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Defined(); got != tt.want {
				t.Errorf("Defined() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKind_Next(t *testing.T) {
	tests := []struct {
		name string
		e    Kind
		want Kind
	}{
		{
			"Kind1",
			Kind1,
			Kind2,
		},
		{
			"Kind2",
			Kind2,
			Kind1,
		},
		{
			"undefined",
			-1,
			Kind1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Next(); got != tt.want {
				t.Errorf("Next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKind_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Kind
		wantErr bool
	}{
		{
			"Kind1",
			"Kind1",
			Kind1,
			false,
		},
		{
			"Kind2",
			"Kind2",
			Kind2,
			false,
		},
		{
			"empty",
			"",
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Kind
			_, err := fmt.Sscan(tt.input, &got)
			if tt.wantErr != (err != nil) {
				t.Error(err)
			}
			if got != tt.want {
				t.Errorf("Scan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKind_String(t *testing.T) {
	tests := []struct {
		name string
		e    Kind
		want string
	}{
		{
			"Kind1",
			Kind1,
			"Kind1",
		},
		{
			"Kind2",
			Kind2,
			"Kind2",
		},
		{
			"undefined",
			-1,
			"Kind(-1)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func ExampleKind_Defined() {
	fmt.Println(Kind(0).Defined())
	fmt.Println(Kind(1).Defined())
	fmt.Println(Kind(2).Defined())

	// Output:
	// true
	// true
	// false
}

func ExampleKind_Scan() {
	var s1, s2 StrKind
	_, _ = fmt.Sscan("Hello World", &s1, &s2)
	fmt.Println(s1)
	fmt.Println(s2)

	// Output:
	// Hello
	// World
}

func ExampleKind_Next() {
	k := Kind(0)
	for {
		fmt.Println(k)
		k = k.Next()
		if k == Kind(0) {
			break
		}
	}

	// Output:
	// Kind1
	// Kind2
}
