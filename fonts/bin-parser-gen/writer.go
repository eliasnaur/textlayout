package binparsergen

import "fmt"

// generated code - writer

// do not perform bounds check
func writeBasicType(sliceName, varName string, size int, offset int) string {
	switch size {
	case bytes1:
		return fmt.Sprintf("%s[%d] = byte(%s)", sliceName, offset, varName)
	case bytes2:
		return fmt.Sprintf("binary.BigEndian.PutUint16(%s[%d:], uint16(%s))", sliceName, offset, varName)
	case bytes4:
		return fmt.Sprintf("binary.BigEndian.PutUint32(%s[%d:], uint32(%s))", sliceName, offset, varName)
	case bytes8:
		return fmt.Sprintf("binary.BigEndian.PutUint64(%s[%d:], uint64(%s))", sliceName, offset, varName)
	default:
		panic("not supported")
	}
}

// append and return
func (fs fixedSizeFields) generateWriter(sliceName, objectName string) string {
	if len(fs) == 0 {
		return ""
	}
	totalSize := fs.size()

	code := fmt.Sprintf(`L := len(%s)
	%s = append(%s, make([]byte, %d)...)
	dst := %s[L:]
	`, sliceName, sliceName, sliceName, totalSize, sliceName)

	pos := 0
	for _, field := range fs {
		var accesVar string
		if field.customConstructor {
			accesVar = fmt.Sprintf("%s.%s.toUint()", objectName, field.field.Name())
		} else {
			accesVar = fmt.Sprintf("%s.%s", objectName, field.field.Name())
		}
		writeCode := writeBasicType("dst", accesVar, field.size, pos)
		code += writeCode + "\n"
		pos += field.size
	}

	return code
}
