package fontconfig

import (
	"fmt"
	"log"
)

var Identity = Matrix{1, 0, 0, 1}

type Object uint16

// The order is part of the cache signature.
const (
	FC_INVALID         Object = iota
	FC_FAMILY                 // String
	FC_FAMILYLANG             // String
	FC_STYLE                  // String
	FC_STYLELANG              // String
	FC_FULLNAME               // String
	FC_FULLNAMELANG           // String
	FC_SLANT                  // Integer
	FC_WEIGHT                 // Range
	FC_WIDTH                  // Range
	FC_SIZE                   // Range
	FC_ASPECT                 // Double
	FC_PIXEL_SIZE             // Double
	FC_SPACING                // Integer
	FC_FOUNDRY                // String
	FC_ANTIALIAS              // Bool
	FC_HINT_STYLE             // Integer
	FC_HINTING                // Bool
	FC_VERTICAL_LAYOUT        // Bool
	FC_AUTOHINT               // Bool
	FC_GLOBAL_ADVANCE         // Bool
	FC_FILE                   // String
	FC_INDEX                  // Integer
	FC_RASTERIZER             // String
	FC_OUTLINE                // Bool
	FC_SCALABLE               // Bool
	FC_DPI                    // Double
	FC_RGBA                   // Integer
	FC_SCALE                  // Double
	FC_MINSPACE               // Bool
	FC_CHARWIDTH              // Integer
	FC_CHAR_HEIGHT            // Integer
	FC_MATRIX                 // Matrix
	FC_CHARSET                // CharSet
	FC_LANG                   // LangSet
	FC_FONTVERSION            // Integer
	FC_CAPABILITY             // String
	FC_FONTFORMAT             // String
	FC_EMBOLDEN               // Bool
	FC_EMBEDDED_BITMAP        // Bool
	FC_DECORATIVE             // Bool
	FC_LCD_FILTER             // Integer
	FC_NAMELANG               // String
	FC_FONT_FEATURES          // String
	FC_PRGNAME                // String
	FC_HASH                   // String
	FC_POSTSCRIPT_NAME        // String
	FC_COLOR                  // Bool
	FC_SYMBOL                 // Bool
	FC_FONT_VARIATIONS        // String
	FC_VARIABLE               // Bool
	FC_FONT_HAS_HINT          // Bool
	FC_ORDER                  // Integer
	FirstCustomObject
)

// Bool is a tri-state boolean (see the associated constants)
type Bool uint8

const (
	FcFalse    Bool = iota // common `false`
	FcTrue                 // common `true`
	FcDontCare             // unspecified
)

func (b Bool) String() string {
	switch b {
	case FcFalse:
		return "false"
	case FcTrue:
		return "true"
	case FcDontCare:
		return "dont-care"
	default:
		return fmt.Sprintf("bool <%d>", b)
	}
}

type Range struct {
	Begin, End float64
}

func FcRangePromote(v Float) Range {
	return Range{Begin: float64(v), End: float64(v)}
}

// returns true if a is inside b
func (a Range) isInRange(b Range) bool {
	return a.Begin >= b.Begin && a.End <= b.End
}

func FcRangeCompare(op FcOp, a, b Range) bool {
	switch op {
	case FcOpEqual:
		return a.Begin == b.Begin && a.End == b.End
	case FcOpContains, FcOpListing:
		return a.isInRange(b)
	case FcOpNotEqual:
		return a.Begin != b.Begin || a.End != b.End
	case FcOpNotContains:
		return !a.isInRange(b)
	case FcOpLess:
		return a.End < b.Begin
	case FcOpLessEqual:
		return a.End <= b.Begin
	case FcOpMore:
		return a.Begin > b.End
	case FcOpMoreEqual:
		return a.Begin >= b.End
	}
	return false
}

type Matrix struct {
	Xx, Xy, Yx, Yy float64
}

// return a * b
func (a Matrix) Multiply(b Matrix) Matrix {
	var r Matrix
	r.Xx = a.Xx*b.Xx + a.Xy*b.Yx
	r.Xy = a.Xx*b.Xy + a.Xy*b.Yy
	r.Yx = a.Yx*b.Xx + a.Yy*b.Yx
	r.Yy = a.Yx*b.Xy + a.Yy*b.Yy
	return r
}

// Hasher mey be implemented by complex value types,
// for which a custom hash is needed.
// Other type use their string representation.
type Hasher interface {
	Hash() []byte
}

// Value is a sum type for the values
// of the properties of a pattern
type Value interface {
	isValue()
	exprNode // usable as expression node
}

func (Int) isValue()     {}
func (Float) isValue()   {}
func (String) isValue()  {}
func (Bool) isValue()    {}
func (Charset) isValue() {}
func (Langset) isValue() {}
func (Matrix) isValue()  {}
func (Range) isValue()   {}

func (Int) isExpr()     {}
func (Float) isExpr()   {}
func (String) isExpr()  {}
func (Bool) isExpr()    {}
func (Charset) isExpr() {}
func (Langset) isExpr() {}
func (Matrix) isExpr()  {}
func (Range) isExpr()   {}

type Int int

type Float float64

type String string

// validate the basic data types
func (object Object) hasValidType(val Value) bool {
	_, isInt := val.(Int)
	_, isFloat := val.(Float)
	switch object {
	case FC_FAMILY, FC_FAMILYLANG, FC_STYLE, FC_STYLELANG, FC_FULLNAME, FC_FULLNAMELANG, FC_FOUNDRY,
		FC_RASTERIZER, FC_CAPABILITY, FC_NAMELANG, FC_FONT_FEATURES, FC_PRGNAME, FC_HASH, FC_POSTSCRIPT_NAME,
		FC_FONTFORMAT, FC_FILE, FC_FONT_VARIATIONS: // string
		_, isString := val.(String)
		return isString
	case FC_ORDER, FC_SLANT, FC_SPACING, FC_HINT_STYLE, FC_RGBA, FC_INDEX,
		FC_CHARWIDTH, FC_LCD_FILTER, FC_FONTVERSION, FC_CHAR_HEIGHT: // integer
		return isInt
	case FC_WEIGHT, FC_WIDTH, FC_SIZE: // range
		_, isRange := val.(Range)
		return isInt || isFloat || isRange
	case FC_ASPECT, FC_PIXEL_SIZE, FC_SCALE, FC_DPI: // float
		return isInt || isFloat
	case FC_ANTIALIAS, FC_HINTING, FC_VERTICAL_LAYOUT, FC_AUTOHINT, FC_GLOBAL_ADVANCE, FC_OUTLINE, FC_SCALABLE,
		FC_MINSPACE, FC_EMBOLDEN, FC_COLOR, FC_SYMBOL, FC_VARIABLE, FC_FONT_HAS_HINT, FC_EMBEDDED_BITMAP, FC_DECORATIVE: // bool
		_, isBool := val.(Bool)
		return isBool
	case FC_MATRIX: // Matrix
		_, isMatrix := val.(Matrix)
		return isMatrix
	case FC_CHARSET: // CharSet
		_, isCharSet := val.(Charset)
		return isCharSet
	case FC_LANG: // LangSet
		_, isLangSet := val.(Langset)
		_, isString := val.(String)
		return isLangSet || isString
	default:
		// no validation
		return true
	}
}

// Compares two values. Integers and Doubles are compared as numbers; otherwise
// the two values have to be the same type to be considered equal. Strings are
// compared ignoring case.
func valueEqual(va, vb Value) bool {
	if v, ok := va.(Int); ok {
		va = Float(v)
	}
	if v, ok := vb.(Int); ok {
		vb = Float(v)
	}

	switch va := va.(type) {
	case nil:
		return vb == nil
	case Float:
		if vb, ok := vb.(Float); ok {
			return va == vb
		}
	case String:
		if vb, ok := vb.(String); ok {
			return cmpIgnoreCase(string(va), string(vb)) == 0
		}
	case Bool:
		if vb, ok := vb.(Bool); ok {
			return va == vb
		}
	case Matrix:
		if vb, ok := vb.(Matrix); ok {
			return va == vb
		}
	case Charset:
		if vb, ok := vb.(Charset); ok {
			return FcCharsetEqual(va, vb)
		}
	case Langset:
		if vb, ok := vb.(Langset); ok {
			return langsetEqual(va, vb)
		}
	case Range:
		if vb, ok := vb.(Range); ok {
			return va.isInRange(vb)
		}
	}
	return false
}

type valueElt struct {
	Value   Value          `json:"v,omitempty"`
	Binding FcValueBinding `json:"b,omitempty"`
}

func (v valueElt) hash() []byte {
	if withHash, ok := v.Value.(Hasher); ok {
		return withHash.Hash()
	}
	return []byte(fmt.Sprintf("%v", v.Value))
}

type FcValueBinding uint8

const (
	FcValueBindingWeak FcValueBinding = iota
	FcValueBindingStrong
	FcValueBindingSame
)

type valueList []valueElt

func (vs valueList) Hash() []byte {
	var hash []byte
	for _, v := range vs {
		hash = append(hash, v.hash()...)
	}
	return hash
}

func (l valueList) prepend(v ...valueElt) valueList {
	l = append(l, make(valueList, len(v))...)
	copy(l[len(v):], l)
	copy(l, v)
	return l
}

// returns a deep copy
func (l valueList) duplicate() valueList {
	// TODO: check the pointer types
	return append(valueList(nil), l...)
}

// insert `newList` into head, begining at `position`.
// If `appendMode` is true, `newList` is inserted just after `position`
// else, `newList` is inserted just before `position`.
// If position == -1, `newList` is inserted at the end or at the begining (depending on `appendMode`)
// `table` is updated for family objects.
// `newList` elements are also typecheked: false is returned if the types are invalid
func (head *valueList) insert(position int, appendMode bool, newList valueList,
	object Object, table *familyTable) bool {

	// Make sure the stored type is valid for built-in objects
	for _, l := range newList {
		if !object.hasValidType(l.Value) {
			log.Printf("fontconfig: pattern object %s does not accept value %v", object, l.Value)
			return false
		}
	}

	if object == FC_FAMILY && table != nil {
		table.add(newList)
	}

	sameBinding := FcValueBindingWeak
	if position != -1 {
		sameBinding = (*head)[position].Binding
	}

	for i, v := range newList {
		if v.Binding == FcValueBindingSame {
			newList[i].Binding = sameBinding
		}
	}

	var cutoff int
	if appendMode {
		if position != -1 {
			cutoff = position + 1
		} else {
			cutoff = len(*head)
		}
	} else {
		if position != -1 {
			cutoff = position
		} else {
			cutoff = 0
		}
	}

	tmp := append(*head, make(valueList, len(newList))...) // allocate
	copy(tmp[cutoff+len(newList):], (*head)[cutoff:])      // make room for newList
	copy(tmp[cutoff:], newList)                            // insert newList
	*head = tmp
	return true
}

// remove the item at `position`
func (head *valueList) del(position int, object Object, table *familyTable) {
	if object == FC_FAMILY && table != nil {
		table.del((*head)[position].Value.(String))
	}

	copy((*head)[position:], (*head)[position+1:])
	(*head)[len((*head))-1] = valueElt{}
	(*head) = (*head)[:len((*head))-1]
}
