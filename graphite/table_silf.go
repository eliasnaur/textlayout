package graphite

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/benoitkugler/textlayout/fonts/binaryreader"
)

type TableSilf []silfSubtable

type silfSubtableHeaderV3 struct {
	RuleVersion   uint32 // Version of stack-machine language used in rules
	PassOffset    uint16 // offset of oPasses[0] relative to start of sub-table
	PseudosOffset uint16 // offset of pMaps[0] relative to start of sub-table
}

type silfSubtablePart1 struct {
	MaxGlyphID   uint16 // Maximum valid glyph ID (including line-break & pseudo-glyphs)
	ExtraAscent  int16  // Em-units to be added to the font’s ascent
	ExtraDescent int16  // Em-units to be added to the font’s descent
	NumPasses    byte   // Number of rendering description passes
	ISubst       byte   // Index of first substitution pass
	IPos         byte   // Index of first Positioning pass
	IJust        byte   // Index of first Justification pass
	IBidi        byte   // Index of first pass after the bidi pass(must be <= iPos); 0xFF implies no bidi pass
	// Bit 0: True (1) if there is any start-, end-, or cross-line contextualization
	// Bit 1: True (1) if cross-line contextualization can be ignored for optimization
	// Bits 2-4: space contextual flags
	// Bit 5: automatic collision fixing
	Flags          byte
	MaxPreContext  byte // Max range for preceding cross-line-boundary contextualization
	MaxPostContext byte // Max range for following cross-line-boundary contextualization

	AttrPseudo         byte // Glyph attribute number that is used for actual glyph ID for a pseudo glyph
	AttrBreakWeight    byte // Glyph attribute number of breakweight attribute
	AttrDirectionality byte // Glyph attribute number for directionality attribute
	AttrMirroring      byte // Glyph attribute number for mirror.glyph (mirror.isEncoded comes directly after)
	AttrSkipPasses     byte // Glyph attribute of bitmap indicating key glyphs for pass optimization
	NumJLevels         byte // Number of justification levels; 0 if no justification
}

type silfSubtablePart2 struct {
	NumLigComp     uint16 // Number of initial glyph attributes that represent ligature components
	NumUserDefn    byte   // Number of user-defined slot attributes
	MaxCompPerLig  byte   // Maximum number of components per ligature
	Direction      byte   // Supported direction(s)
	AttrCollisions byte   // Glyph attribute number for collision.flags attribute (several more collision attrs come after it...)

	_ [3]byte // reserved
}

func parseTableSilf(data []byte) (TableSilf, error) {
	r := binaryreader.NewReader(data)

	version_, err := r.Uint32()
	if err != nil {
		return nil, fmt.Errorf("invalid table Silf: %s", err)
	}
	version := uint16(version_ >> 16)
	if version < 2 {
		return nil, fmt.Errorf("invalid table Silf version: %x", version)
	}
	if version >= 3 {
		r.Skip(4)
	}

	numSub, err := r.Uint16()
	if err != nil {
		return nil, fmt.Errorf("invalid table Silf: %s", err)
	}
	r.Skip(2) // reserved

	offsets, err := r.Uint32s(int(numSub))
	if err != nil {
		return nil, fmt.Errorf("invalid table Silf: %s", err)
	}

	out := make([]silfSubtable, numSub)
	for i, offset := range offsets {
		out[i], err = parseSubtableSilf(data, offset, version)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

type pseudoMap struct {
	Unicode rune
	NPseudo GID
}

type silfSubtable struct {
	justificationLevels []JustificationLevel // length NumJLevels
	scriptTags          []uint32
	classMap            classMap
	passes              []silfPass
	critFeatures        []uint16
	pseudoMap           []pseudoMap
	lbGID               uint16
	silfSubtablePart1
	silfSubtablePart2
}

func (s *silfSubtable) findPdseudoGlyph(r rune) GID {
	if s == nil {
		return 0
	}
	for _, rec := range s.pseudoMap {
		if rec.Unicode == r {
			return rec.NPseudo
		}
	}
	return 0
}

type binSearchHeader struct {
	NumRecord uint16
	_         [3]uint16 // ignored
}

func parseSubtableSilf(data []byte, offset uint32, version uint16) (out silfSubtable, err error) {
	if len(data) < int(offset) {
		return out, fmt.Errorf("invalid Silf subtable offset: %d", offset)
	}
	data = data[offset:] // needed for passes
	r := binaryreader.NewReader(data)

	var headerv3 silfSubtableHeaderV3
	if version >= 3 {
		err = r.ReadStruct(&headerv3)
		if err != nil {
			return out, fmt.Errorf("invalid Silf subtable: %s", err)
		}
	}
	err = r.ReadStruct(&out.silfSubtablePart1)
	if err != nil {
		return out, fmt.Errorf("invalid Silf subtable: %s", err)
	}

	out.justificationLevels = make([]JustificationLevel, out.silfSubtablePart1.NumJLevels) // allocation guarded by the uint8 constraint
	err = r.ReadStruct(out.justificationLevels)
	if err != nil {
		return out, fmt.Errorf("invalid Silf subtable: %s", err)
	}

	err = r.ReadStruct(&out.silfSubtablePart2)
	if err != nil {
		return out, fmt.Errorf("invalid Silf subtable: %s", err)
	}

	numCritFeatures, err := r.Byte() // Number of critical features
	if err != nil {
		return out, fmt.Errorf("invalid Silf subtable: %s", err)
	}
	out.critFeatures, err = r.Uint16s(int(numCritFeatures))
	if err != nil {
		return out, fmt.Errorf("invalid Silf subtable: %s", err)
	}
	r.Skip(1) // byte reserved

	numScriptTag, err := r.Byte() // Number of scripts this subtable supports
	if err != nil {
		return out, fmt.Errorf("invalid Silf subtable: %s", err)
	}
	out.scriptTags, err = r.Uint32s(int(numScriptTag)) //  Array of numScriptTag script tags
	if err != nil {
		return out, fmt.Errorf("invalid Silf subtable: %s", err)
	}

	out.lbGID, err = r.Uint16() // Glyph ID for line-break psuedo-glyph
	if err != nil {
		return out, fmt.Errorf("invalid Silf subtable: %s", err)
	}

	oPasses, err := r.Uint32s(int(out.silfSubtablePart1.NumPasses) + 1) // Offets to passes relative to the start of this subtable
	if err != nil {
		return out, fmt.Errorf("invalid Silf subtable: %s", err)
	}

	var mapsHeader binSearchHeader
	err = r.ReadStruct(&mapsHeader)
	if err != nil {
		return out, fmt.Errorf("invalid Silf subtable: %s", err)
	}

	out.pseudoMap = make([]pseudoMap, mapsHeader.NumRecord) // Mappings between Unicode and pseudo-glyphs in order of Unicode
	err = r.ReadStruct(out.pseudoMap)
	if err != nil {
		return out, fmt.Errorf("invalid Silf subtable: %s", err)
	}

	out.classMap, err = parseGraphiteClassMap(r.Data(), version)
	if err != nil {
		return out, err
	}

	out.passes = make([]silfPass, out.silfSubtablePart1.NumPasses)
	for i := range out.passes {
		offset := oPasses[i]
		out.passes[i], err = parseSilfPass(data, offset)
		if err != nil {
			return out, err
		}
	}

	return out, nil
}

type JustificationLevel struct {
	attrStretch byte    //  Glyph attribute number for justify.X.stretch
	attrShrink  byte    //  Glyph attribute number for justify.X.shrink
	attrStep    byte    //  Glyph attribute number for justify.X.step
	attrWeight  byte    //  Glyph attribute number for justify.X.weight
	runto       byte    //  Which level starts the next stage
	_           [3]byte // reserved
}

type classMap struct {
	// numClass  uint16
	// numLinear uint16
	// oClass    []uint32      // Array of numClass + 1 offsets to class arrays from the beginning of the class map

	glyphs  [][]GID               // Glyphs for linear classes (length numLinear)
	lookups []graphiteLookupClass // An array of numClass – numLinear lookups
}

func (cl classMap) numClasses() uint16 { return uint16(len(cl.glyphs) + len(cl.lookups)) }

func (cl classMap) getClassGlyph(cid uint16, index int) GID {
	if int(cid) < len(cl.glyphs) { // linear
		if glyphs := cl.glyphs[cid]; index < len(glyphs) {
			return glyphs[index]
		}
	} else if lookupIndex := int(cid) - len(cl.glyphs); lookupIndex < len(cl.lookups) {
		lookup := cl.lookups[lookupIndex]
		for _, entry := range lookup {
			if int(entry.Index) == index {
				return entry.Glyph
			}
		}
	}
	return 0
}

// returns -1 if not found
func (cl classMap) findClassIndex(cid uint16, gid GID) int {
	if int(cid) < len(cl.glyphs) { // linear
		for index, g := range cl.glyphs[cid] {
			if g == gid {
				return index
			}
		}
	} else if lookupIndex := int(cid) - len(cl.glyphs); lookupIndex < len(cl.lookups) {
		lookup := cl.lookups[lookupIndex]
		for _, entry := range lookup {
			if entry.Glyph == gid {
				return int(entry.Index)
			}
		}
	}
	return -1
}

// data starts at the class map
func parseGraphiteClassMap(data []byte, version uint16) (out classMap, err error) {
	r := binaryreader.NewReader(data)
	if len(data) < 4 {
		return out, errors.New("invalid Silf Class Map (EOF)")
	}
	numClass, _ := r.Uint16()  // Number of replacement classes
	numLinear, _ := r.Uint16() // Number of linearly stored replacement classes

	var offsets []uint32
	if version >= 4 {
		offsets, err = r.Uint32s(int(numClass) + 1)
		if err != nil {
			return out, fmt.Errorf("invalid Silf Class Map (with long offsets): %s", err)
		}
	} else {
		var slice []byte
		slice, err = r.FixedSizes(int(numClass)+1, 2)
		if err != nil {
			return out, fmt.Errorf("invalid Silf Class Map (with short offsets): %s", err)
		}
		offsets = make([]uint32, int(numClass)+1)
		for i := range offsets {
			offsets[i] = uint32(binary.BigEndian.Uint16(slice[2*i:]))
		}
	}

	if numClass < numLinear {
		return out, fmt.Errorf("invalid Silf Class Map (%d < %d)", numClass, numLinear)
	}

	out.glyphs = make([][]GID, numLinear)
	for i := range out.glyphs {
		start, end := offsets[i], offsets[i+1]
		if start > end {
			return out, fmt.Errorf("invalid Silf Class Map offset (%d > %d)", start, end)
		}

		out.glyphs[i] = make([]GID, (end-start)/2)
		r.SetPos(int(start))
		_ = r.ReadStruct(out.glyphs[i])
	}

	out.lookups = make([]graphiteLookupClass, numClass-numLinear)

	for i := range out.lookups {
		offset := int(offsets[int(numLinear)+i])
		r.SetPos(offset) // delay error checking
		out.lookups[i], err = parseGraphiteLookupClass(r)
		if err != nil {
			return out, err
		}
	}

	return out, nil
}

type graphiteLookupPair struct {
	Glyph GID
	Index uint16
}

type graphiteLookupClass []graphiteLookupPair

// r is positionned at the start
func parseGraphiteLookupClass(r *binaryreader.Reader) (graphiteLookupClass, error) {
	numsIDS, err := r.Uint16()
	if err != nil {
		return nil, fmt.Errorf("invalid Silf Lookup Class: %s", err)
	}
	r.Skip(6)
	out := make(graphiteLookupClass, numsIDS)
	err = r.ReadStruct(out)
	if err != nil {
		return nil, fmt.Errorf("invalid Silf Lookup Class: %s", err)
	}
	return out, nil
}

type silfPassHeader struct {
	// Bits 0-2: collision fixing max loop
	// Bits 3-4: auto kerning
	// Bit 5: flip direction 5.0 – added
	Flags           byte
	MaxRuleLoop     byte   // MaxRuleLoop for this pass
	MaxRuleContext  byte   // Number of slots of input needed to run this pass
	MaxBackup       byte   // Number of slots by which the following pass needs to trail this pass (ie, the maximum this pass is allowed to back up)
	NumRules        uint16 // Number of action code blocks
	FsmOffset       uint16 // offset to numRows relative to the beginning of the SIL_Pass block 2.0 – inserted ; 3.0 – use for fsmOffset
	PcCode          uint32 // Offset to start of pass constraint code from start of subtable (*passConstraints[0]*) 2.0 - added
	RcCode          uint32 // Offset to start of rule constraint code from start of subtable (*ruleConstraints[0]*)
	ACode           uint32 // Offset to start of action code relative to start of subtable (*actions[0]*)
	ODebug          uint32 // Offset to debug arrays (*dActions[0]*); equals 0 if debug stripped
	NumRows         uint16 // Number of FSM states
	NumTransitional uint16 // Number of transitional states in the FSM (length of *states* matrix)
	NumSuccess      uint16 // Number of success states in the FSM (size of *oRuleMap* array)
	NumColumns      uint16 // Number of FSM columns; 0 means no FSM
}

type passRange struct {
	FirstId GID    // First Glyph id in the range
	LastId  GID    // Last Glyph id in the range
	ColId   uint16 // Column index for this range
}

func parseSilfPass(data []byte, offset uint32) (out silfPass, err error) {
	r, err := binaryreader.NewReaderAt(data, offset)
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass offset: %s", err)
	}

	err = r.ReadStruct(&out.silfPassHeader) // length was checked
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass header: %s", err)
	}

	numRange, err := r.Uint16()
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}
	r.Skip(6)
	out.Ranges = make([]passRange, numRange)
	err = r.ReadStruct(out.Ranges)
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}

	oRuleMap, err := r.Uint16s(int(out.NumSuccess) + 1)
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}
	ruleMapSlice, err := r.Uint16s(int(oRuleMap[len(oRuleMap)-1]))
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}
	out.ruleMap = make([][]uint16, out.NumSuccess)
	for i := range out.ruleMap {
		start, end := oRuleMap[i], oRuleMap[i+1]
		if start > end || int(end) > len(ruleMapSlice) {
			continue
		}
		out.ruleMap[i] = ruleMapSlice[start:end]
	}

	minRulePreContext, err := r.Byte() // Minimum number of items in any rule’s context before the first modified rule item
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}
	maxRulePreContext, err := r.Byte() // Maximum number of items in any rule’s context before the first modified rule item
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}
	if maxRulePreContext < minRulePreContext {
		return out, fmt.Errorf("invalid Silf Pass subtable pre-context rule: (%d < %d)", maxRulePreContext, minRulePreContext)
	}
	out.startStates, err = r.Int16s(int(maxRulePreContext-minRulePreContext) + 1)
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}

	out.ruleSortKeys, err = r.Uint16s(int(out.NumRules))
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}

	out.rulePreContext, err = r.FixedSizes(int(out.NumRules), 1)
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}

	out.collisionThreshold, err = r.Byte()
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}

	pConstraint, err := r.Uint16()
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}

	oConstraints, err := r.Uint16s(int(out.NumRules) + 1)
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}

	oActions, err := r.Uint16s(int(out.NumRules) + 1)
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}

	transitions, err := r.Uint16s(int(out.NumTransitional) * int(out.NumColumns))
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}
	out.stateTransitions = make([][]uint16, out.NumTransitional)
	for i := range out.stateTransitions {
		out.stateTransitions[i] = transitions[i*int(out.NumColumns) : (i+1)*int(out.NumColumns)]
	}

	r.Skip(1)

	out.passConstraints, err = r.FixedSizes(int(pConstraint), 1)
	if err != nil {
		return out, fmt.Errorf("invalid Silf Pass subtable: %s", err)
	}

	out.ruleConstraints = make([][]byte, out.NumRules)
	ruleConstraintsSlice := r.Data()
	for i := range out.ruleConstraints {
		offsetStart, offsetEnd := oConstraints[i], oConstraints[i+1]
		if offsetEnd <= offsetStart {
			continue
		}
		if int(offsetEnd) > len(ruleConstraintsSlice) {
			return out, fmt.Errorf("invalid Silf Pass subtable rule constraint offset: %d", offsetEnd)
		}
		out.ruleConstraints[i] = ruleConstraintsSlice[offsetStart:offsetEnd]
	}

	out.actions = make([][]byte, out.NumRules)
	actionsSlice := ruleConstraintsSlice[oConstraints[len(oConstraints)-1]:]
	for i := range out.actions {
		offsetStart, offsetEnd := oActions[i], oActions[i+1]
		if offsetEnd <= offsetStart {
			continue
		}
		if int(offsetEnd) > len(actionsSlice) {
			return out, fmt.Errorf("invalid Silf Pass subtable rule constraint offset: %d", offsetEnd)
		}
		out.actions[i] = actionsSlice[offsetStart:offsetEnd]
	}
	return out, nil
}

func (silf *silfSubtable) runGraphite(seg *segment, firstPass, lastPass uint8, doBidi bool) bool {
	maxSize := len(seg.charinfo) * MAX_SEG_GROWTH_FACTOR
	fmt.Println(maxSize)
	// SlotMap            map(*seg, m_dir, maxSize);
	// FiniteStateMachine fsm(map, seg.getFace().logger());
	// vm::Machine        m(map);
	lbidi := silf.IBidi

	if lastPass == 0 {
		if firstPass == lastPass && lbidi == 0xFF {
			return true
		}
		lastPass = silf.NumPasses
	}
	if (firstPass < lbidi || (doBidi && firstPass == lbidi)) && (lastPass >= lbidi || (doBidi && lastPass+1 == lbidi)) {
		lastPass++
	} else {
		lbidi = 0xFF
	}

	for i := firstPass; i < lastPass; i++ {
		// bidi and mirroring
		if i == lbidi {
			// #if !defined GRAPHITE2_NTRACING
			//             if (dbgout)
			//             {
			//                 *dbgout << json::item << json::object
			// //							<< "pindex" << i   // for debugging
			//                             << "id"     << -1
			//                             << "slotsdir" << (seg.currdir() ? "rtl" : "ltr")
			//                             << "passdir" << (m_dir & 1 ? "rtl" : "ltr")
			//                             << "slots"  << json::array;
			//                 seg.positionSlots(0, 0, 0, seg.currdir());
			//                 for(Slot * s = seg.first(); s; s = s.next())
			//                     *dbgout     << dslot(seg, s);
			//                 *dbgout         << json::close
			//                             << "rules"  << json::array << json::close
			//                             << json::close;
			//             }
			// #endif
			if seg.currdir() != (silf.Direction&1 != 0) {
				seg.reverseSlots()
			}
			if mirror := silf.AttrMirroring; mirror != 0 && (seg.dir&3) == 3 {
				seg.doMirror(mirror)
			}
			i--
			lbidi = lastPass
			lastPass--
			continue
		}

		// #if !defined GRAPHITE2_NTRACING
		//         if (dbgout)
		//         {
		//             *dbgout << json::item << json::object
		// //						<< "pindex" << i   // for debugging
		//                         << "id"     << i+1
		//                         << "slotsdir" << (seg.currdir() ? "rtl" : "ltr")
		//                         << "passdir" << ((silf.Direction & 1) ^ silf.passes[i].isReverseDir() ? "rtl" : "ltr")
		//                         << "slots"  << json::array;
		//             seg.positionSlots(0, 0, 0, seg.currdir());
		//             for(Slot * s = seg.first(); s; s = s.next())
		//                 *dbgout     << dslot(seg, s);
		//             *dbgout         << json::close;
		//         }
		// #endif

		// TODO:
		// // test whether to reorder, prepare for positioning
		// reverse := (lbidi == 0xFF) && (seg.currdir() != ((silf.Direction&1 != 0) != silf.passes[i].isReverseDir()))
		// if (i >= 32 || (seg.passBits&(1<<i)) == 0 || silf.passes[i].collisionLoops() != 0) &&
		// 	!silf.passes[i].runGraphite(m, fsm, reverse) {
		// 	return false
		// }
		// // only subsitution passes can change segment length, cached subsegments are short for their text
		// if m.status() != vm__Machine__finished || (len(seg.charinfo) != 0 && len(seg.charinfo) > maxSize) {
		// 	return false
		// }
	}
	return true
}
