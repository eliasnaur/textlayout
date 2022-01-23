package binparsergen

import (
	"testing"
)

func Test_importSource(t *testing.T) {
	err := generateParser()
	if err != nil {
		t.Fatal(err)
	}
}
