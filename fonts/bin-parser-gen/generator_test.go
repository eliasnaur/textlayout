package binparsergen

import (
	"testing"
)

func Test_importSource(t *testing.T) {
	name, _, err := importSource("test-package")
	if err != nil {
		t.Fatal(err)
	}
	if name != "testpackage" {
		t.Fatalf("unexpected package name %s", name)
	}
}

func Test_generateParser(t *testing.T) {
	err := Generate("test-package")
	if err != nil {
		t.Fatal(err)
	}
}
