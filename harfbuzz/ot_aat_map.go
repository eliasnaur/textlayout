package harfbuzz

import (
	"sort"

	tt "github.com/benoitkugler/textlayout/fonts/truetype"
)

// ported from harfbuzz/src/hb-aat-map.cc, hb-att-map.hh Copyright © 2018  Google, Inc. Behdad Esfahbod

type hb_aat_map_t struct {
	chain_flags []Mask
}

type aat_feature_info_t struct {
	type_        hb_aat_layout_feature_type_t
	setting      hb_aat_layout_feature_selector_t
	is_exclusive bool

	// 	 /* compares type & setting only, not is_exclusive flag or seq number */
	// 	 int cmp (const aat_feature_info_t& f) const
	// 	 {
	// 	   return (f.type != type) ? (f.type < type ? -1 : 1) :
	// 		  (f.setting != setting) ? (f.setting < setting ? -1 : 1) : 0;
	// 	 }
	//    };
}

func (fi aat_feature_info_t) key() uint32 {
	return uint32(fi.type_<<16) | uint32(fi.setting)
}

const selMask = ^hb_aat_layout_feature_selector_t(1)

func cmpAATFeatureInfo(a, b aat_feature_info_t) bool {
	if a.type_ != b.type_ {
		return a.type_ < b.type_
	}
	if !a.is_exclusive && (a.setting&selMask) != (b.setting&selMask) {
		return a.setting < b.setting
	}
	return false
}

type hb_aat_map_builder_t struct {
	face     Face
	features []aat_feature_info_t // sorted by (type_, setting) after compilation
}

// binary search into `features`, comparing type_ and setting only
func (mb *hb_aat_map_builder_t) hasFeature(info aat_feature_info_t) bool {
	key := info.key()
	for i, j := 0, len(mb.features); i < j; {
		h := i + (j-i)/2
		entry := mb.features[h].key()
		if key < entry {
			j = h
		} else if entry < key {
			i = h + 1
		} else {
			return true
		}
	}
	return false
}

func (mapper *hb_aat_map_builder_t) compileMap(map_ *hb_aat_map_t) {
	morx := mapper.face.getMorxTable()
	for _, chain := range morx {
		map_.chain_flags = append(map_.chain_flags, mapper.compileMorxFlag(chain))
	}

	// TODO: for now we dont support deprecated mort table
	// mort := mapper.face.table.mort
	// if mort.has_data() {
	// 	mort.compile_flags(mapper, map_)
	// 	return
	// }
}

func (mapper *hb_aat_map_builder_t) compileMorxFlag(chain tt.MorxChain) Mask {
	flags := chain.DefaultFlags

	for _, feature := range chain.Features {
		type_, setting := feature.Type, feature.Setting

	retry:
		// Check whether this type_/setting pair was requested in the map, and if so, apply its flags.
		// (The search here only looks at the type_ and setting fields of feature_info_t.)
		info := aat_feature_info_t{type_, setting, false}
		if mapper.hasFeature(info) {
			flags &= feature.DisableFlags
			flags |= feature.EnableFlags
		} else if type_ == HB_AAT_LAYOUT_FEATURE_TYPE_LETTER_CASE && setting == HB_AAT_LAYOUT_FEATURE_SELECTOR_SMALL_CAPS {
			/* Deprecated. https://github.com/harfbuzz/harfbuzz/issues/1342 */
			type_ = HB_AAT_LAYOUT_FEATURE_TYPE_LOWER_CASE
			setting = HB_AAT_LAYOUT_FEATURE_SELECTOR_LOWER_CASE_SMALL_CAPS
			goto retry
		}
	}
	return flags
}

func (mb *hb_aat_map_builder_t) add_feature(tag hb_tag_t, value uint32) {
	feat := mb.face.getFeatTable()
	if len(feat) == 0 {
		return
	}

	if tag == newTag('a', 'a', 'l', 't') {
		if fn := feat.GetFeature(HB_AAT_LAYOUT_FEATURE_TYPE_CHARACTER_ALTERNATIVES); fn == nil || len(fn.Settings) == 0 {
			return
		}
		info := aat_feature_info_t{
			type_:        HB_AAT_LAYOUT_FEATURE_TYPE_CHARACTER_ALTERNATIVES,
			setting:      hb_aat_layout_feature_selector_t(value),
			is_exclusive: true,
		}
		mb.features = append(mb.features, info)
		return
	}

	mapping := aatLayoutFindFeatureMapping(tag)
	if mapping == nil {
		return
	}

	feature := feat.GetFeature(mapping.aatFeatureType)
	if feature == nil || len(feature.Settings) == 0 {
		/* Special case: compileMorxFlag() will fall back to the deprecated version of
		 * small-caps if necessary, so we need to check for that possibility.
		 * https://github.com/harfbuzz/harfbuzz/issues/2307 */
		if mapping.aatFeatureType == HB_AAT_LAYOUT_FEATURE_TYPE_LOWER_CASE &&
			mapping.selectorToEnable == HB_AAT_LAYOUT_FEATURE_SELECTOR_LOWER_CASE_SMALL_CAPS {
			feature = feat.GetFeature(HB_AAT_LAYOUT_FEATURE_TYPE_LETTER_CASE)
			if feature == nil || len(feature.Settings) == 0 {
				return
			}
		} else {
			return
		}
	}

	var info aat_feature_info_t
	info.type_ = mapping.aatFeatureType
	if value != 0 {
		info.setting = mapping.selectorToEnable
	} else {
		info.setting = mapping.selectorToDisable
	}
	info.is_exclusive = feature.IsExclusive()
	mb.features = append(mb.features, info)
}

func (mb *hb_aat_map_builder_t) compile(m *hb_aat_map_t) {
	// sort features and merge duplicates
	if len(mb.features) != 0 {
		sort.SliceStable(mb.features, func(i, j int) bool {
			return cmpAATFeatureInfo(mb.features[i], mb.features[j])
		})
		j := 0
		for i := 1; i < len(mb.features); i++ {
			/* Nonexclusive feature selectors come in even/odd pairs to turn a setting on/off
			* respectively, so we mask out the low-order bit when checking for "duplicates"
			* (selectors referring to the same feature setting) here. */
			if mb.features[i].type_ != mb.features[j].type_ ||
				(!mb.features[i].is_exclusive && ((mb.features[i].setting & selMask) != (mb.features[j].setting & selMask))) {
				j++
				mb.features[j] = mb.features[i]
			}
		}
		mb.features = mb.features[:j+1]
	}

	mb.compileMap(m)
}
