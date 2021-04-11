package main

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Generator of the function to prohibit certain vowel sequences.
//
// It creates ``_preprocessTextVowelConstraints``, which inserts dotted
// circles into sequences prohibited by the USE script development spec.
// This function should be used as the ``preprocess_text`` of an
// ``hb_ot_complex_shaper_t``.

func aggregateVowelData(scriptsClasses map[string][]rune, constraintsRunes [][]rune) (map[string]*constraintSet, map[string]rune) {
	scripts := map[rune]string{}
	scriptOrder := map[string]rune{} // first rune in the script
	for s, rs := range scriptsClasses {
		start := rs[0]
		for _, r := range rs {
			if start > r {
				start = r
			}
			scripts[r] = s
		}
		scriptOrder[s] = start
	}

	constraints := map[string]*constraintSet{}
	for _, constraint := range constraintsRunes {
		script := scripts[constraint[0]]
		if cs := constraints[script]; cs != nil {
			cs.add(constraint)
		} else {
			constraints[script] = newConstraintSet(constraint)
		}
	}
	if len(constraints) == 0 {
		check(errors.New("no cluster constraints found"))
	}

	return constraints, scriptOrder
}

func generateVowelConstraints(scriptsClasses map[string][]rune, constraintsRunes [][]rune, w io.Writer) {
	constraints, scriptOrder := aggregateVowelData(scriptsClasses, constraintsRunes)

	fmt.Fprintln(w, `
	package harfbuzz

	// Code generated by unicodedata/generate/main.go DO NOT EDIT.
	
	`)

	fmt.Fprintln(w, "func outputDottedCircle (buffer *Buffer) {")
	fmt.Fprintln(w, "	buffer.outputRune(0x25CC)")
	fmt.Fprintln(w, " 	buffer.prev().resetContinutation()")
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "func outputWithDottedCircle (buffer *Buffer) {")
	fmt.Fprintln(w, " 	outputDottedCircle (buffer);")
	fmt.Fprintln(w, " 	buffer.nextGlyph ();")
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w)

	fmt.Fprintln(w, `func preprocessTextVowelConstraints (buffer *Buffer) {
		if (buffer.Flags & DoNotinsertDottedCircle) != 0 { return }
		
		/* UGLY UGLY UGLY business of adding dotted-circle in the middle of
		* vowel-sequences that look like another vowel. Data for each script
		* collected from the USE script development spec.
		*
		* https://github.com/harfbuzz/harfbuzz/issues/1019
		*/
		processed := false;
		buffer.clearOutput ();
		count := len(buffer.Info);
		switch  buffer.Props.Script {
	`)

	var sortedConstraints []string // sorted (constraints.items (), key=lambda s_c: script_order[s_c[0]])
	for k := range constraints {
		sortedConstraints = append(sortedConstraints, k)
	}
	sort.Slice(sortedConstraints, func(i, j int) bool {
		return scriptOrder[sortedConstraints[i]] < scriptOrder[sortedConstraints[j]]
	})
	for _, script := range sortedConstraints {
		constraint := constraints[script]
		fmt.Fprintf(w, "case language.%s:\n", script)
		fmt.Fprintf(w, `	for buffer.idx = 0; buffer.idx + 1 < count; {
								matched := false
								%s
								buffer.nextGlyph ()
								if (matched) { outputWithDottedCircle (buffer) }
		      				}
						  processed = true;
		`, constraint)
	}
	fmt.Fprintln(w, `}
					if (processed) {
						if (buffer.idx < count){
							buffer.nextGlyph ()
						}
						buffer.swapBuffers ()
					}
				}`)
}

// a set of prohibited code point sequences.
// Either a list or a dictionary. As a list of code points, it
// represents a prohibited code point sequence. As a dictionary,
// it represents a set of prohibited sequences, where each item
// represents the set of prohibited sequences starting with the
// key (a code point) concatenated with any of the values
// (ConstraintSets).
type constraintSet struct {
	dict   map[rune]*constraintSet
	list   []rune
	isList bool
}

// compare a and b, with b truncated to length(a)
// if it is larger
func runesEqual(a, b []rune) bool {
	if len(b) > len(a) {
		b = b[:len(a)]
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func newConstraintSet(l []rune) *constraintSet { return &constraintSet{isList: true, list: l} }

// Add a constraint to this set."""
func (cs *constraintSet) add(constraint []rune) {
	if len(constraint) == 0 {
		return
	}
	first := constraint[0]
	rest := constraint[1:]
	if cs.isList {
		if runesEqual(constraint, cs.list) {
			cs.list = constraint
		} else if !runesEqual(cs.list, constraint) {
			cs.isList = false
			cs.dict = map[rune]*constraintSet{cs.list[0]: newConstraintSet(cs.list[1:])}
		}
	} else {
		if firstCs, has := cs.dict[first]; has {
			firstCs.add(rest)
		} else {
			cs.dict[first] = newConstraintSet(rest)
		}
	}
}

func (cs *constraintSet) String() string {
	return cs.string(0)
}

func (cs *constraintSet) string(index int) string {
	var s []string
	if cs.isList {
		if len(cs.list) == 0 {
			if index != 2 {
				check(errors.New("cannot use `matched` for this constraint; the general case has not been implemented"))
			}
			s = append(s, "matched = true\n")
		} else if len(cs.list) == 1 {
			if index != 1 {
				check(errors.New("cannot use `matched` for this constraint; the general case has not been implemented"))
			}
			s = append(s, fmt.Sprintf("matched = 0x%04X == buffer.cur(%d).codepoint\n", cs.list[0], index))
		} else {
			s = append(s, fmt.Sprintf("if 0x%04X == buffer.cur(%d).codepoint &&\n", cs.list[0], index))
			if index != 0 {
				s = append(s, fmt.Sprintf("buffer.idx + %d < count &&\n", index+1))
			}
			for i, cp := range cs.list[1:] {
				close := " &&"
				if i+1 == len(cs.list)-1 {
					close = ""
				}
				s = append(s, fmt.Sprintf("0x%04X == buffer.cur(%d).codepoint%s ", cp, index+i+1, close))
			}
			s = append(s, "{\n")
			for i := 0; i < index; i++ {
				s = append(s, "buffer.nextGlyph ()\n")
			}
			s = append(s, "matched = true\n")
			s = append(s, "}\n")
		}
	} else {
		s = append(s, fmt.Sprintf("switch buffer.cur(%d).codepoint { \n", index))
		cases := map[string]map[rune]bool{}
		var keysSorted []rune
		for r := range cs.dict {
			keysSorted = append(keysSorted, r)
		}
		sortRunes(keysSorted)

		for _, first := range keysSorted {
			rest := cs.dict[first]
			str := rest.string(index + 1)
			set := cases[str]
			if set == nil {
				set = make(map[rune]bool)
			}
			set[first] = true
			cases[str] = set
		}
		var keys []string
		for k := range cases {
			keys = append(keys, k)
		}
		sortRuneSet := func(s map[rune]bool) []rune {
			runes := make([]rune, 0, len(s))
			for r := range s {
				runes = append(runes, r)
			}
			sortRunes(runes)
			return runes
		}
		sort.Slice(keys, func(i, j int) bool {
			si, sj := cases[keys[i]], cases[keys[j]]
			return sortRuneSet(si)[0] < sortRuneSet(sj)[0]
		})

		for _, body := range keys {
			labels := cases[body]
			for i, cp := range sortRuneSet(labels) {
				s = append(s, " ")
				end := ""
				if i%4 == 3 {
					end = "\n"
				}
				s = append(s, fmt.Sprintf("case 0x%04X:%s", cp, end))
				if len(labels)%4 != 0 {
					s = append(s, "\n")
				}
				s = append(s, body)
			}
		}
		s = append(s, "}\n")
	}
	return strings.Join(s, "")
}
