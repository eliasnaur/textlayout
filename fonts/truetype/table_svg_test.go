package truetype

import (
	"os"
	"strings"
	"testing"

	"github.com/benoitkugler/textlayout/fonts"
)

func TestParseSVG(t *testing.T) {
	f, err := os.Open("testdata/chromacheck-svg.ttf")
	if err != nil {
		t.Fatal(err)
	}
	pr, err := NewFontParser(f)
	if err != nil {
		t.Fatal(err)
	}
	table, err := pr.GetRawTable(tagSVG)
	if err != nil {
		t.Fatal(err)
	}

	svg, err := parseTableSVG(table)
	if err != nil {
		t.Fatal(err)
	}

	if _, has := svg.glyphData(0); has {
		t.Fatal("unexpected svg data")
	}
	data, has := svg.glyphData(1)
	if !has {
		t.Fatal("missing svg data")
	}
	source := string(data.Source)
	if !strings.HasPrefix(source, "<?xml") ||
		!strings.HasSuffix(source, "</svg>") ||
		!strings.Contains(source, `id="glyph1"`) {
		t.Fatalf("unexpected svg data; %s", source)
	}
}

func TestGlyphDataSVG(t *testing.T) {
	font := loadFont(t, "testdata/chromacheck-svg.ttf")
	data := font.GlyphData(1, 0, 0)
	if _, ok := data.(fonts.GlyphSVG); !ok {
		t.Fatalf("unexpected glyph data %v", data)
	}
}
