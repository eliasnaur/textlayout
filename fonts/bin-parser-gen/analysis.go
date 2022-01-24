package binparsergen

import (
	"fmt"
	"go/types"
	"reflect"
)

// one or many field whose parsing (or writting)
// is grouped to reduce length checks and allocations
type structChunk interface {
	generateParser(fieldIndex int, srcVar, returnVars, offsetExpression string) string

	generateAppender(fieldIndex int, srcVar, dstSlice string) string
}

// either a basic type or a struct with fixed sized fields
type fixedSizeType interface {
	name() string
	size() int

	// how to implement
	// dstVar = parse(dataSrcVar[offset:])
	generateParser(dstVar, srcSlice string, offset int) string

	// how to implement
	// put srcVar into dstSlice[offset:]
	generateWriter(srcVar, dstSlice string, offset int) string
}

// check is the underlying type as fixed size;
// return nil if not
func isFixedSize(ty types.Type) fixedSizeType {
	// first check for custom constructor
	// if present, only the constructor type matters
	layout := hasConstructor(ty)
	if layout != 0 { // overide underlying basic info
		return withConstructor{name_: ty.(*types.Named).Obj().Name(), size_: layout}
	}

	switch underlying := ty.Underlying().(type) {
	case *types.Basic:
		if L, ok := getBinaryLayout(underlying); ok {
			return basicType{Basic: *underlying, binaryLayout: L}
		}
	case *types.Struct:
		named, ok := ty.(*types.Named)
		if !ok {
			panic("anonymous struct not supported")
		}

		fields, ok := fixedSizeFromStruct(underlying)
		if ok {
			return fixedSizeStruct{
				type_: underlying,
				name_: named.Obj().Name(),
				size_: fields.size(),
			}
		}
	case *types.Array:
		panic("array not supported yet")

	}
	return nil
}

type withConstructor struct {
	name_ string
	size_ int
}

func (wc withConstructor) name() string {
	return wc.name_
}

func (wc withConstructor) size() int {
	return wc.size_
}

type basicType struct {
	types.Basic

	binaryLayout int
}

func (bt basicType) name() string {
	return bt.Basic.Name()
}

func (bt basicType) size() int { return bt.binaryLayout }

// a struct with fixed size
type fixedSizeStruct struct {
	type_ *types.Struct // underlying type

	name_ string
	size_ int
}

func (f fixedSizeStruct) name() string {
	return f.name_
}

func (f fixedSizeStruct) size() int {
	return f.size_
}

// how the type is written as binary
type fixedSizeField struct {
	field *types.Var

	type_ fixedSizeType
}

const (
	bytes1 = 1 // byte, int8
	bytes2 = 2 // int16, uint16
	bytes4 = 4 // uint32
	bytes8 = 8 // uint32
)

func getBinaryLayout(t *types.Basic) (int, bool) {
	switch t.Kind() {
	case types.Bool, types.Int8, types.Uint8:
		return bytes1, true
	case types.Int16, types.Uint16:
		return bytes2, true
	case types.Int32, types.Uint32, types.Float32:
		return bytes4, true
	case types.Int64, types.Uint64, types.Float64:
		return bytes8, true
	default:
		return 0, false
	}
}

// return the new binary layout, or 0
// if always returns 0 if ty is not a *types.Named
func hasConstructor(ty types.Type) int {
	// a type with a method is a named type
	named, ok := ty.(*types.Named)
	if !ok {
		return 0
	}

	for i := 0; i < named.NumMethods(); i++ {
		meth := named.Method(i)
		if meth.Name() == "fromUint" {
			arg := meth.Type().(*types.Signature).Params().At(0).Type().Underlying()
			if basic, ok := arg.(*types.Basic); ok {
				if layout, ok := getBinaryLayout(basic); ok {
					return layout
				}
			}
		}
	}

	return 0
}

type fixedSizeFields []fixedSizeField

// return true is all fields are with fixed size
func fixedSizeFromStruct(st *types.Struct) (fixedSizeFields, bool) {
	var fixedSize fixedSizeFields
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)

		if ft := isFixedSize(field.Type()); ft != nil {
			fixedSize = append(fixedSize, fixedSizeField{field: field, type_: ft})
		} else {
			return fixedSize, false
		}
	}
	return fixedSize, true
}

// returns the total size needed by the fields
func (fs fixedSizeFields) size() int {
	totalSize := 0
	for _, field := range fs {
		totalSize += int(field.type_.size())
	}
	return totalSize
}

type arrayField struct {
	field *types.Var

	sizeLen int
	element fixedSizeType
}

func newSliceField(field *types.Var, tag string) (arrayField, bool) {
	if fieldType, ok := field.Type().Underlying().(*types.Slice); ok {
		var af arrayField
		af.field = field

		fieldElement := isFixedSize(fieldType.Elem())
		if fieldElement == nil {
			panic("slice of variable length element are not supported")
		}

		af.element = fieldElement

		tag := reflect.StructTag(tag)
		switch tag.Get("len-size") {
		case "16":
			af.sizeLen = bytes2
		case "32":
			af.sizeLen = bytes4
		case "64":
			af.sizeLen = bytes8
		default:
			panic(fmt.Sprintf("missing tag 'len-size' for %s", field.String()))
		}

		return af, true
	}

	return arrayField{}, false
}

func analyseStruct(st *types.Struct) (out []structChunk) {
	var fixedSize fixedSizeFields
	for i := 0; i < st.NumFields(); i++ {
		field, tag := st.Field(i), st.Tag(i)

		if ft := isFixedSize(field.Type()); ft != nil {
			fixedSize = append(fixedSize, fixedSizeField{field: field, type_: ft})
			continue
		}

		// close the current fixedSize array
		if len(fixedSize) != 0 {
			out = append(out, fixedSize)
			fixedSize = nil
		}

		// and try for slice
		af, ok := newSliceField(field, tag)
		if ok {
			out = append(out, af)
			continue
		}

		panic(fmt.Sprintf("unsupported field in struct %s", field))
	}

	// close the current fixedSize array
	if len(fixedSize) != 0 {
		out = append(out, fixedSize)
	}

	return out
}
