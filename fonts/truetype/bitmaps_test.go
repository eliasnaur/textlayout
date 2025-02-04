package truetype

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/benoitkugler/textlayout/fonts"
)

func TestSbix(t *testing.T) {
	for _, filename := range []string{
		"testdata/ToyFeat.ttf",
		"testdata/ToySbix.ttf",
	} {
		file, err := os.Open(filename)
		if err != nil {
			t.Fatalf("Failed to open %q: %s\n", filename, err)
		}

		font, err := NewFontParser(file)
		if err != nil {
			t.Fatalf("Parse(%q) err = %q, want nil", filename, err)
		}

		ng, err := font.NumGlyphs()
		if err != nil {
			t.Fatal(err)
		}

		gs, err := font.sbixTable(ng)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println("Number of strkes:", len(gs.strikes))

		for gid := GID(0); gid < fonts.GID(ng); gid++ {
			for _, strike := range gs.strikes {
				g := strike.getGlyph(gid, 0)
				if g.isNil() {
					continue
				}
				if _, ok := g.glyphExtents(); !ok {
					t.Error(filename, gid)
				}
			}
		}

		file.Close()
	}
}

func TestCblc(t *testing.T) {
	for _, filename := range []string{
		"testdata/ToyCBLC1.ttf",
		"testdata/ToyCBLC2.ttf",
		"testdata/NotoColorEmoji.ttf",
	} {
		file, err := os.Open(filename)
		if err != nil {
			t.Fatalf("Failed to open %q: %s\n", filename, err)
		}

		pr, err := NewFontParser(file)
		if err != nil {
			t.Fatalf("Parse(%q) err = %q, want nil", filename, err)
		}

		cmaps, err := pr.CmapTable()
		if err != nil {
			t.Fatal(err)
		}

		gs, err := pr.colorBitmapTable()
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println("Number of strikes:", len(gs))
		for _, strike := range gs {
			fmt.Println("number of subtables:", len(strike.subTables))
		}

		head, err := pr.loadHeadTable()
		if err != nil {
			t.Fatal(err)
		}

		file.Close()

		font := Font{bitmap: gs, upem: head.Upem()}
		cmap, _ := cmaps.BestEncoding()
		iter := cmap.Iter()
		for iter.Next() {
			_, gid := iter.Char()
			font.getExtentsFromCBDT(gid, 94, 94)
		}
	}
}

func TestEblc(t *testing.T) {
	for _, filename := range []string{
		"testdata/mry_KacstQurn.ttf",
		"testdata/IBM3161-bitmap.otb",
	} {
		file, err := os.Open(filename)
		if err != nil {
			t.Fatal(filename, err)
		}

		font, err := NewFontParser(file)
		if err != nil {
			t.Fatal(filename, err)
		}

		gs, err := font.grayBitmapTable()
		if err != nil {
			t.Fatal(err)
		}

		for _, strike := range gs {
			fmt.Println(len(strike.subTables))
			strike.subTables = nil // not to flood the terminal
			fmt.Println(strike)
		}
		file.Close()
	}
}

func TestAppleBitmap(t *testing.T) {
	filename := "testdata/Gacha_9.dfont"
	file, err := os.Open(filename)
	if err != nil {
		t.Fatal(filename, err)
	}
	defer file.Close()

	fonts, err := NewFontParsers(file)
	if err != nil {
		t.Fatal(err)
	}

	font := fonts[0]

	gs, err := font.appleBitmapTable()
	if err != nil {
		t.Fatal(err)
	}

	for _, strike := range gs {
		fmt.Println(len(strike.subTables))
	}
}

func TestSize(t *testing.T) {
	expectedSizes := [][]fonts.BitmapSize{
		{
			{Height: 300, Width: 300, XPpem: 300, YPpem: 300},
		},
		{
			{Height: 127, Width: 136, XPpem: 109, YPpem: 109},
		},
		{
			{Height: 128, Width: 136, XPpem: 109, YPpem: 109},
		},
		{
			{Height: 128, Width: 117, XPpem: 94, YPpem: 94},
		},
		{
			{Height: 128, Width: 136, XPpem: 109, YPpem: 109},
		},
		{
			{Height: 33, Width: 8, XPpem: 16, YPpem: 16},
			{Height: 44, Width: 10, XPpem: 21, YPpem: 21},
		},
		{
			{Height: 16, Width: 15, XPpem: 16, YPpem: 16},
		},
		{
			{Height: 9, Width: 6, XPpem: 9, YPpem: 9}, // freetype actually gives a width of 0, which is suspicious
		},
	}
	for i, filename := range []string{
		"testdata/ToyFeat.ttf",
		"testdata/ToySbix.ttf",
		"testdata/ToyCBLC1.ttf",
		"testdata/ToyCBLC2.ttf",
		"testdata/NotoColorEmoji.ttf",
		"testdata/mry_KacstQurn.ttf",
		"testdata/IBM3161-bitmap.otb",
		"testdata/Gacha_9.dfont",
	} {
		file, err := os.Open(filename)
		if err != nil {
			t.Fatal(filename, err)
		}

		fonts, err := Load(file)
		if err != nil {
			t.Fatal(filename, err)
		}

		font := fonts[0].(*Font)
		got := font.LoadBitmaps()
		if !reflect.DeepEqual(got, expectedSizes[i]) {
			t.Fatalf("font %s, expected %v got %v", filename, expectedSizes[i], got)
		}

		file.Close()
	}
}
