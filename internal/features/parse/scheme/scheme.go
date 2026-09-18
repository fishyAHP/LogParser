package scheme

type DataType uint8

const (
	Invalid DataType = iota
	StringType
	IntType
	BoolType
	FloatType
	TimeType
	ArrayType
	MapType
	ObjectType
)

type Field struct {
	Name         string
	FieldType    DataType
	ObjectFields []Field
}

type Scheme struct {
	separator  rune
	parameters []Field
}

func (s *Scheme) Validate() bool {
	return true
}
