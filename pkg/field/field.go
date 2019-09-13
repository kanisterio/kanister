package field

// Field is a tuple with a string key and a value of any type
type Field interface {
	Key() string
	Value() interface{}
}

// Fieldser is a collection of fields
type Fieldser interface {
	Fields() []Field
}

// Fields is an alias for Fieldser to make the interface friendlier. It seems
// easier to talk about fields than fieldser(s)
type Fields = Fieldser

// New creates a new field collection with given key and value
func New(key string, value interface{}) Fields {
	return Add(nil, key, value)
}

// Add returns a collection with all the fields in s plus a new field with the
// given key and value. Duplicates are not eliminated.
func Add(s Fields, key string, value interface{}) Fields {
	return newField(s, key, value)
}

// M contains fields with unique keys. Used to facilitate adding multiple
// "fields" to a Fields collection
type M = map[string]interface{}

// addMap adds the entries in m to s as Field(s). The map key is used as the
// Field.Key() and the corresponding value as Field.Value()
func addMap(s Fields, m M) Fields {
	for k, v := range m {
		s = Add(s, k, v)
	}
	return s
}
