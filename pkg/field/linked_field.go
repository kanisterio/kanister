package field

import (
	"fmt"
	"strings"
)

// Fields are exposed as an interface instead of directly as pointer to this
// struct. That enforces the immutability of fields outside this package. Once
// a field is created, its contents cannot be changed or overwritten outside
// this package by simply assigning one field to another, via
// f1 = f2 or *fp1 = *fp2.
type field struct {
	key   string
	value interface{}
}

var _ fmt.Stringer = field{}

func (f field) Key() string {
	return f.key
}

func (f field) Value() interface{} {
	return f.value
}

func (f field) String() string {
	return fmt.Sprintf("%q:%q", f.key, f.value)
}

type linkedField struct {
	field
	prev *linkedField
}

var _ fmt.Stringer = (*linkedField)(nil)

func newField(prev Fields, key string, value interface{}) *linkedField {
	return &linkedField{prev: asLinkedFields(prev), field: field{key: key, value: value}}
}

func (f *linkedField) Fields() []Field {
	return f.fields(0)
}

func (f *linkedField) fields(n int) []Field {
	if f != nil {
		return append(f.prev.fields(n+1), f.field)
	}
	return make([]Field, 0, n)
}

func (f *linkedField) String() string {
	var b strings.Builder
	b.WriteByte('[')
	if f != nil {
		f.buildString(&b)
	}
	b.WriteByte(']')
	return b.String()
}

func (f *linkedField) buildString(b *strings.Builder) {
	if f.prev != nil {
		f.prev.buildString(b)
		b.WriteByte(',')
	}
	fmt.Fprintf(b, "%q:%q", f.key, f.value)
}

func asLinkedFields(fs Fields) *linkedField {
	if f, ok := fs.(*linkedField); ok {
		return f
	}
	return toLinkedFields(nil, fs)
}

func toLinkedFields(prev *linkedField, fs Fields) *linkedField {
	if fs == nil {
		return prev
	}
	for _, o := range fs.Fields() {
		prev = &linkedField{prev: prev, field: field{key: o.Key(), value: o.Value()}}
	}
	return prev
}
