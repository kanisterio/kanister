package field_test

import (
	"context"
	"fmt"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/field"
)

type FieldSuite struct{}

var _ = Suite(&FieldSuite{})

func ExampleNew() {
	f := field.New("foo", "bar")
	fmt.Print(f)
	// Output: ["foo":"bar"]
}

func ExampleAdd() {
	f := field.New("foo", "bar")
	f = field.Add(f, "baz", "x")
	fmt.Print(f)
	// Output: ["foo":"bar","baz":"x"]
}

type M = field.M

func ExampleContext() {
	ctx := field.Context(context.Background(), "foo", "bar")
	fmt.Print(field.FromContext(ctx))
	// Output: ["foo":"bar"]
}

func ExampleAddMapToContext() {
	ctx := field.AddMapToContext(context.Background(), M{"foo": "bar"})
	fmt.Print(field.FromContext(ctx))
	// Output: ["foo":"bar"]
}

func ExampleAddMapToContext_multiple() {
	ctx := field.AddMapToContext(context.Background(), M{"foo": "bar", "x": "y"})
	// Output is not specified because the order of the fields in 'M' is non-deterministic
	fmt.Print(field.FromContext(ctx))
}
