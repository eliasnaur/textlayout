package binparsergen

import "fmt"

// generated code - parser

// return the code needed to check the length of a byte slice
func checkLength(sliceName, returnVars string, minLength int) string {
	return fmt.Sprintf(`if L := len(%s); L < %d {
		return %s, fmt.Errorf("EOF: expected length: %d, got %%d", L)
	}
	`, sliceName, minLength, returnVars, minLength)
}

// do not perform bounds check
func readBasicType(sliceName string, size int, offset int) string {
	switch size {
	case bytes1:
		return fmt.Sprintf("%s[%d]", sliceName, offset)
	case bytes2:
		return fmt.Sprintf("binary.BigEndian.Uint16(%s[%d:%d])", sliceName, offset, offset+2)
	case bytes4:
		return fmt.Sprintf("binary.BigEndian.Uint32(%s[%d:%d])", sliceName, offset, offset+4)
	case bytes8:
		return fmt.Sprintf("binary.BigEndian.Uint64(%s[%d:%d])", sliceName, offset, offset+8)
	default:
		panic("not supported")
	}
}

func (fs fixedSizeFields) generateParser(sliceName string, returnVars string) string {
	if len(fs) == 0 {
		return ""
	}

	code := checkLength(sliceName, returnVars, fs.size())

	pos := 0
	for _, field := range fs {
		readCode := readBasicType(sliceName, field.size, pos)

		if field.customConstructor {
			code += fmt.Sprintf("out.%s.fromUint(%s)\n", field.field.Name(), readCode)
		} else {
			constructor := field.field.Type().String()
			code += fmt.Sprintf("out.%s = %s(%s)\n", field.field.Name(), constructor, readCode)
		}

		pos += field.size
	}

	return code
}
