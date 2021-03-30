package harfbuzz

// ported from harfbuzz/src/hb-ot-shape-fallback.cc Copyright © 2011,2012 Google, Inc. Behdad Esfahbod

const (
	combiningClassAttachedBelowLeft  = 200
	combiningClassAttachedBelow      = 202
	combiningClassAttachedAbove      = 214
	combiningClassAttachedAboveRight = 216
	combiningClassBelowLeft          = 218
	combiningClassBelow              = 220
	combiningClassBelowRight         = 222
	combiningClassLeft               = 224
	combiningClassRight              = 226
	combiningClassAboveLeft          = 228
	combiningClassAbove              = 230
	combiningClassAboveRight         = 232
	combiningClassDoubleBelow        = 233
	combiningClassDoubleAbove        = 234
)

func recategorizeCombiningClass(u rune, klass uint8) uint8 {
	if klass >= 200 {
		return klass
	}

	/* Thai / Lao need some per-character work. */
	if (u & ^0xFF) == 0x0E00 {
		if klass == 0 {
			switch u {
			case 0x0E31, 0x0E34, 0x0E35, 0x0E36, 0x0E37, 0x0E47, 0x0E4C, 0x0E4D, 0x0E4E:
				klass = combiningClassAboveRight
			case 0x0EB1, 0x0EB4, 0x0EB5, 0x0EB6, 0x0EB7, 0x0EBB, 0x0ECC, 0x0ECD:
				klass = combiningClassAbove
			case 0x0EBC:
				klass = combiningClassBelow
			}
		} else {
			/* Thai virama is below-right */
			if u == 0x0E3A {
				klass = combiningClassBelowRight
			}
		}
	}

	switch klass {

	/* Hebrew */

	case 10, /* sheva */
		11, /* hataf segol */
		12, /* hataf patah */
		13, /* hataf qamats */
		14, /* hiriq */
		15, /* tsere */
		16, /* segol */
		17, /* patah */
		18, /* qamats */
		20, /* qubuts */
		22: /* meteg */
		return combiningClassBelow

	case 23: /* rafe */
		return combiningClassAttachedAbove

	case 24: /* shin dot */
		return combiningClassAboveRight

	case 25, /* sin dot */
		19: /* holam */
		return combiningClassAboveLeft

	case 26: /* point varika */
		return combiningClassAbove

	case 21: /* dagesh */

	/* Arabic and Syriac */

	case 27, /* fathatan */
		28, /* dammatan */
		30, /* fatha */
		31, /* damma */
		33, /* shadda */
		34, /* sukun */
		35, /* superscript alef */
		36: /* superscript alaph */
		return combiningClassAbove

	case 29, /* kasratan */
		32: /* kasra */
		return combiningClassBelow

	/* Thai */

	case 103: /* sara u / sara uu */
		return combiningClassBelowRight

	case 107: /* mai */
		return combiningClassAboveRight

	/* Lao */

	case 118: /* sign u / sign uu */
		return combiningClassBelow

	case 122: /* mai */
		return combiningClassAbove

	/* Tibetan */

	case 129: /* sign aa */
		return combiningClassBelow

	case 130: /* sign i*/
		return combiningClassAbove

	case 132: /* sign u */
		return combiningClassBelow

	}

	return klass
}

func fallbackMarkPositionRecategorizeMarks(buffer *Buffer) {
	for i, info := range buffer.Info {
		if info.unicode.generalCategory() == NonSpacingMark {
			combiningClass := info.getModifiedCombiningClass()
			combiningClass = recategorizeCombiningClass(info.codepoint, combiningClass)
			buffer.Info[i].setModifiedCombiningClass(combiningClass)
		}
	}
}

func zeroMarkAdvances(buffer *Buffer, start, end int, adjustOffsetsWhenZeroing bool) {
	info := buffer.Info
	for i := start; i < end; i++ {
		if info[i].unicode.generalCategory() != NonSpacingMark {
			continue
		}
		if adjustOffsetsWhenZeroing {
			buffer.Pos[i].XOffset -= buffer.Pos[i].XAdvance
			buffer.Pos[i].YOffset -= buffer.Pos[i].YAdvance
		}
		buffer.Pos[i].XAdvance = 0
		buffer.Pos[i].YAdvance = 0
	}
}

func positionMark(font *Font, buffer *Buffer, baseExtents *glyphExtents,
	i int, combiningClass uint8) {
	markExtents, ok := font.getGlyphExtents(buffer.Info[i].Glyph)
	if !ok {
		return
	}

	yGap := font.YScale / 16

	pos := &buffer.Pos[i]
	pos.XOffset = 0
	pos.YOffset = 0

	// we don't position LEFT and RIGHT marks.

	// X positioning
	switch combiningClass {
	case combiningClassAttachedBelowLeft, combiningClassBelowLeft, combiningClassAboveLeft:
		/* Left align. */
		pos.XOffset += baseExtents.XBearing - markExtents.XBearing

	case combiningClassAttachedAboveRight, combiningClassBelowRight, combiningClassAboveRight:
		/* Right align. */
		pos.XOffset += baseExtents.XBearing + baseExtents.Width - markExtents.Width - markExtents.XBearing
	case combiningClassDoubleBelow, combiningClassDoubleAbove:
		if buffer.Props.Direction == LeftToRight {
			pos.XOffset += baseExtents.XBearing + baseExtents.Width - markExtents.Width/2 - markExtents.XBearing
			break
		} else if buffer.Props.Direction == RightToLeft {
			pos.XOffset += baseExtents.XBearing - markExtents.Width/2 - markExtents.XBearing
			break
		}
		fallthrough
	case combiningClassAttachedBelow, combiningClassAttachedAbove, combiningClassBelow, combiningClassAbove:
		fallthrough
	default:
		/* Center align. */
		pos.XOffset += baseExtents.XBearing + (baseExtents.Width-markExtents.Width)/2 - markExtents.XBearing
	}

	/* Y positioning */
	switch combiningClass {
	case combiningClassDoubleBelow, combiningClassBelowLeft, combiningClassBelow, combiningClassBelowRight:
		/* Add gap, fall-through. */
		baseExtents.Height -= yGap
		fallthrough

	case combiningClassAttachedBelowLeft, combiningClassAttachedBelow:
		pos.YOffset = baseExtents.YBearing + baseExtents.Height - markExtents.YBearing
		/* Never shift up "below" marks. */
		if (yGap > 0) == (pos.YOffset > 0) {
			baseExtents.Height -= pos.YOffset
			pos.YOffset = 0
		}
		baseExtents.Height += markExtents.Height

	case combiningClassDoubleAbove, combiningClassAboveLeft, combiningClassAbove, combiningClassAboveRight:
		/* Add gap, fall-through. */
		baseExtents.YBearing += yGap
		baseExtents.Height -= yGap
		fallthrough
	case combiningClassAttachedAbove, combiningClassAttachedAboveRight:
		pos.YOffset = baseExtents.YBearing - (markExtents.YBearing + markExtents.Height)
		/* Don't shift down "above" marks too much. */
		if (yGap > 0) != (pos.YOffset > 0) {
			correction := -pos.YOffset / 2
			baseExtents.YBearing += correction
			baseExtents.Height -= correction
			pos.YOffset += correction
		}
		baseExtents.YBearing -= markExtents.Height
		baseExtents.Height += markExtents.Height
	}
}

func positionAroundBase(plan *otShapePlan, font *Font, buffer *Buffer,
	base, end int, adjustOffsetsWhenZeroing bool) {
	buffer.unsafeToBreak(base, end)

	baseExtents, ok := font.getGlyphExtents(buffer.Info[base].Glyph)
	if !ok {
		// if extents don't work, zero marks and go home.
		zeroMarkAdvances(buffer, base+1, end, adjustOffsetsWhenZeroing)
		return
	}
	baseExtents.YBearing += buffer.Pos[base].YOffset
	/* Use horizontal advance for horizontal positioning.
	* Generally a better idea.  Also works for zero-ink glyphs.  See:
	* https://github.com/harfbuzz/harfbuzz/issues/1532 */
	baseExtents.XBearing = 0
	baseExtents.Width = font.getGlyphHAdvance(buffer.Info[base].Glyph)

	ligID := buffer.Info[base].getLigID()
	numLigComponents := int32(buffer.Info[base].getLigNumComps())

	var xOffset, yOffset Position
	if buffer.Props.Direction.isForward() {
		xOffset -= buffer.Pos[base].XAdvance
		yOffset -= buffer.Pos[base].YAdvance
	}

	var horizDir Direction
	componentExtents := baseExtents
	lastLigComponent := int32(-1)
	lastCombiningClass := uint8(255)
	clusterExtents := baseExtents
	info := buffer.Info
	for i := base + 1; i < end; i++ {
		if info[i].getModifiedCombiningClass() != 0 {
			if numLigComponents > 1 {
				thisLigID := info[i].getLigID()
				thisLigComponent := int32(info[i].getLigComp() - 1)
				// conditions for attaching to the last component.
				if ligID == 0 || ligID != thisLigID || thisLigComponent >= numLigComponents {
					thisLigComponent = numLigComponents - 1
				}
				if lastLigComponent != thisLigComponent {
					lastLigComponent = thisLigComponent
					lastCombiningClass = 255
					componentExtents = baseExtents
					if horizDir == 0 {
						if plan.props.Direction.isHorizontal() {
							horizDir = plan.props.Direction
						} else {
							horizDir = getHorizontalDirection(plan.props.Script)
						}
					}
					if horizDir == LeftToRight {
						componentExtents.XBearing += (thisLigComponent * componentExtents.Width) / numLigComponents
					} else {
						componentExtents.XBearing += ((numLigComponents - 1 - thisLigComponent) * componentExtents.Width) / numLigComponents
					}
					componentExtents.Width /= numLigComponents
				}
			}

			thisCombiningClass := info[i].getModifiedCombiningClass()
			if lastCombiningClass != thisCombiningClass {
				lastCombiningClass = thisCombiningClass
				clusterExtents = componentExtents
			}

			positionMark(font, buffer, &clusterExtents, i, thisCombiningClass)

			buffer.Pos[i].XAdvance = 0
			buffer.Pos[i].YAdvance = 0
			buffer.Pos[i].XOffset += xOffset
			buffer.Pos[i].YOffset += yOffset

		} else {
			if buffer.Props.Direction.isForward() {
				xOffset -= buffer.Pos[i].XAdvance
				yOffset -= buffer.Pos[i].YAdvance
			} else {
				xOffset += buffer.Pos[i].XAdvance
				yOffset += buffer.Pos[i].YAdvance
			}
		}
	}
}

func positionCluster(plan *otShapePlan, font *Font, buffer *Buffer,
	start, end int, adjustOffsetsWhenZeroing bool) {
	if end-start < 2 {
		return
	}

	// find the base glyph
	info := buffer.Info
	for i := start; i < end; i++ {
		if !info[i].isUnicodeMark() {
			// find mark glyphs
			var j int
			for j = i + 1; j < end; j++ {
				if !info[j].isUnicodeMark() {
					break
				}
			}

			positionAroundBase(plan, font, buffer, i, j, adjustOffsetsWhenZeroing)

			i = j - 1
		}
	}
}

func fallbackMarkPosition(plan *otShapePlan, font *Font, buffer *Buffer,
	adjustOffsetsWhenZeroing bool) {
	var start int
	info := buffer.Info
	for i := 1; i < len(info); i++ {
		if !info[i].isUnicodeMark() {
			positionCluster(plan, font, buffer, start, i, adjustOffsetsWhenZeroing)
			start = i
		}
	}
	positionCluster(plan, font, buffer, start, len(info), adjustOffsetsWhenZeroing)
}

//  /* Performs font-assisted kerning. */
// func (plan *hb_ot_shape_plan_t) _hb_ot_shape_fallback_kern (font *Font,  buffer * Buffer)  {

//  #ifndef HB_DISABLE_DEPRECATED
//    if (HB_DIRECTION_IS_HORIZONTAL (buffer.props.direction) ?
// 	   !font.has_glyph_h_kerning_func () :
// 	   !font.has_glyph_v_kerning_func ())
// 	 return;

//    bool reverse = HB_DIRECTION_IS_BACKWARD (buffer.props.direction);

//    if (reverse)
// 	 buffer.reverse ();

//    hb_ot_shape_fallback_kern_driver_t driver (font, buffer);
//    OT::hb_kern_machine_t<hb_ot_shape_fallback_kern_driver_t> machine (driver);
//    machine.kern (font, buffer, plan.kern_mask, false);

//    if (reverse)
// 	 buffer.reverse ();
//  #endif
//  }

// adjusts width of various spaces.
func fallbackSpaces(font *Font, buffer *Buffer) {
	info := buffer.Info
	pos := buffer.Pos
	horizontal := buffer.Props.Direction.isHorizontal()
	for i, inf := range info {
		if !inf.IsUnicodeSpace() || inf.ligated() {
			continue
		}

		spaceType := inf.getUnicodeSpaceFallbackType()

		switch spaceType {
		case notSpace, space: // shouldn't happen
		case spaceEM, spaceEM2, spaceEM3, spaceEM4, spaceEM5, spaceEM6, spaceEM16:
			if horizontal {
				pos[i].XAdvance = +(font.XScale + int32(spaceType)/2) / int32(spaceType)
			} else {
				pos[i].YAdvance = -(font.YScale + int32(spaceType)/2) / int32(spaceType)
			}
		case space4EM18:
			if horizontal {
				pos[i].XAdvance = +font.XScale * 4 / 18
			} else {
				pos[i].YAdvance = -font.YScale * 4 / 18
			}
		case spaceFigure:
			for u := '0'; u <= '9'; u++ {
				if glyph, ok := font.face.GetNominalGlyph(u); ok {
					if horizontal {
						pos[i].XAdvance = font.getGlyphHAdvance(glyph)
					} else {
						pos[i].YAdvance = font.getGlyphVAdvance(glyph)
					}
				}
			}
		case spacePunctuation:
			glyph, ok := font.face.GetNominalGlyph('.')
			if !ok {
				glyph, ok = font.face.GetNominalGlyph(',')
			}
			if ok {
				if horizontal {
					pos[i].XAdvance = font.getGlyphHAdvance(glyph)
				} else {
					pos[i].YAdvance = font.getGlyphVAdvance(glyph)
				}
			}
		case spaceNarrow:
			/* Half-space?
			* Unicode doc https://unicode.org/charts/PDF/U2000.pdf says ~1/4 or 1/5 of EM.
			* However, in my testing, many fonts have their regular space being about that
			* size.  To me, a percentage of the space width makes more sense.  Half is as
			* good as any. */
			if horizontal {
				pos[i].XAdvance /= 2
			} else {
				pos[i].YAdvance /= 2
			}
		}
	}
}
