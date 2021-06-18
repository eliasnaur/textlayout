package truetype

var _ FaceVariable = (*Font)(nil)

// FaceVariable returns the variations for the font,
// or an empty table.
type FaceVariable interface {
	Variations() TableFvar

	// NormalizeVariations should normalize the given design-space coordinates. The minimum and maximum
	// values for the axis are mapped to the interval [-1,1], with the default
	// axis value mapped to 0.
	// This should be a no-op for non-variable fonts.
	NormalizeVariations(coords []float32) []float32
}

// Variation defines a value for a wanted variation axis.
type Variation struct {
	Tag   Tag     // variation-axis identifier tag
	Value float32 // in design units
}

type VarAxis struct {
	Tag     Tag     // Tag identifying the design variation for the axis.
	Minimum float32 // mininum value on the variation axis that the font covers
	Default float32 // default position on the axis
	Maximum float32 // maximum value on the variation axis that the font covers
	flags   uint16  // Axis qualifiers — see details below.
	strid   NameID  // name entry in the font's ‘name’ table
}

type VarInstance struct {
	Coords    []float32 // in design units; length: number of axis
	Subfamily NameID

	PSStringID NameID
}

type TableFvar struct {
	Axis      []VarAxis
	Instances []VarInstance // contains the default instance
}

// IsDefaultInstance returns `true` is `instance` has the same
// coordinates as the default instance.
func (t TableFvar) IsDefaultInstance(it VarInstance) bool {
	for i, c := range it.Coords {
		if c != t.Axis[i].Default {
			return false
		}
	}
	return true
}

// add the default instance if it not already explicitely present
func (t *TableFvar) checkDefaultInstance(names TableName) {
	for _, instance := range t.Instances {
		if t.IsDefaultInstance(instance) {
			return
		}
	}

	// add the default instance
	// choose the subfamily entry
	subFamily := NamePreferredSubfamily
	if v1, v2 := names.getEntry(subFamily); v1 == nil && v2 == nil {
		subFamily = NameFontSubfamily
	}
	defaultInstance := VarInstance{
		Coords:     make([]float32, len(t.Axis)),
		Subfamily:  subFamily,
		PSStringID: NamePostscript,
	}
	for i, axe := range t.Axis {
		defaultInstance.Coords[i] = axe.Default
	}
	t.Instances = append(t.Instances, defaultInstance)
}

// findAxisIndex return the axis for the given tag, by its index, or -1 if not found.
func (t *TableFvar) findAxisIndex(tag Tag) int {
	for i, axis := range t.Axis {
		if axis.Tag == tag {
			return i
		}
	}
	return -1
}

// GetDesignCoordsDefault returns the design coordinates corresponding to the given pairs of axis/value.
// The default value of the axis is used when not specified in the variations.
func (t *TableFvar) GetDesignCoordsDefault(variations []Variation) []float32 {
	designCoords := make([]float32, len(t.Axis))
	// start with default values
	for i, axis := range t.Axis {
		designCoords[i] = axis.Default
	}

	t.GetDesignCoords(variations, designCoords)

	return designCoords
}

// GetDesignCoords updates the design coordinates, with the given pairs of axis/value.
// It will panic if `designCoords` has not the right length.
func (t *TableFvar) GetDesignCoords(variations []Variation, designCoords []float32) {
	for _, variation := range variations {
		index := t.findAxisIndex(variation.Tag)
		if index == -1 {
			continue
		}
		designCoords[index] = variation.Value
	}
}

// normalize based on the [min,def,max] values for the axis to be [-1,0,1].
func (t *TableFvar) normalizeCoordinates(coords []float32) []float32 {
	normalized := make([]float32, len(coords))
	for i, a := range t.Axis {
		coord := coords[i]

		// out of range: clamping
		if coord > a.Maximum {
			coord = a.Maximum
		} else if coord < a.Minimum {
			coord = a.Minimum
		}

		if coord < a.Default {
			normalized[i] = -(coord - a.Default) / (a.Minimum - a.Default)
		} else if coord > a.Default {
			normalized[i] = (coord - a.Default) / (a.Maximum - a.Default)
		} else {
			normalized[i] = 0
		}
	}
	return normalized
}

func (f *Font) Variations() TableFvar {
	return f.fvar
}

// Normalizes the given design-space coordinates. The minimum and maximum
// values for the axis are mapped to the interval [-1,1], with the default
// axis value mapped to 0.
// Any additional scaling defined in the face's `avar` table is also
// applied, as described at https://docs.microsoft.com/en-us/typography/opentype/spec/avar
func (f *Font) NormalizeVariations(coords []float32) []float32 {
	// ported from freetype2

	// Axis normalization is a two-stage process.  First we normalize
	// based on the [min,def,max] values for the axis to be [-1,0,1].
	// Then, if there's an `avar' table, we renormalize this range.
	normalized := f.fvar.normalizeCoordinates(coords)

	// now applying 'avar'
	for i, av := range f.avar {
		for j := 1; j < len(av); j++ {
			previous, pair := av[j-1], av[j]
			if normalized[i] < pair.from {
				normalized[i] =
					previous.to + (normalized[i]-previous.from)*
						(pair.to-previous.to)/(pair.from-previous.from)
				break
			}
		}
	}

	return normalized
}
