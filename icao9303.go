// Package icao9303 transliterates Russian text from Cyrillic to Latin following
// the ICAO Doc 9303 standard used when issuing Russian international passports.
//
// The character mapping is taken from the appendix "Транслитерация
// кириллических знаков (извлечение)" of the MRZ formation algorithm approved by
// Order No. 2113 of the Ministry of Foreign Affairs of the Russian Federation,
// dated 12 February 2020:
//
//	а->a  б->b  в->v  г->g  д->d  е->e  ё->e  ж->zh  з->z  и->i  й->i
//	к->k  л->l  м->m  н->n  о->o  п->p  р->r  с->s  т->t  у->u  ф->f
//	х->kh ц->ts ч->ch ш->sh щ->shch ъ->ie ы->y  ь->(dropped)
//	э->e  ю->iu я->ia
//
// The soft sign (ь) is not listed in the official extract; per ICAO Doc 9303
// practice it has no Latin equivalent and is dropped.
//
// The standard is one-way and lossy: several Cyrillic letters map to the same
// Latin sequence (е/ё/э->e, и/й->i) and the soft sign is dropped entirely. For
// that reason [ToCyrillic] is a best-effort reverse transformation and does not
// guarantee recovery of the original text.
package icao9303

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// _forward holds the base (lowercase) Latin mapping for each lowercase letter of
// the Russian alphabet а..я (U+0430..U+044F), indexed by the offset r-'а'.
//
// Direct array indexing replaces a hash map: lookup is O(1) with no hashing and
// no allocation. The letters Ё/ё lie outside the block and are handled
// separately. The soft sign (ь) maps to an empty string and is dropped.
var _forward = [32]string{
	"a", "b", "v", "g", "d", "e", "zh", "z", "i", "i", "k", "l",
	"m", "n", "o", "p", "r", "s", "t", "u", "f", "kh", "ts", "ch",
	"sh", "shch", "ie", "y", "", "e", "iu", "ia",
}

// foldIndex reports the index of r in _forward, whether r is uppercase, and
// whether r is a Russian letter present in the table.
//
// The literals 'а', 'А', 'я' are compile-time code-point constants (U+0430,
// etc.), so writing a character or its numeric code produces identical machine
// code; the character form is simply more readable.
func foldIndex(r rune) (idx int, upper, ok bool) {
	switch {
	case r >= 'а' && r <= 'я': // U+0430..U+044F, lowercase
		return int(r - 'а'), false, true
	case r >= 'А' && r <= 'Я': // U+0410..U+042F, uppercase
		return int(r - 'А'), true, true
	case r == 'ё':
		return int('е' - 'а'), false, true
	case r == 'Ё':
		return int('е' - 'а'), true, true
	}

	return
}

// ToLatin transliterates Cyrillic text to Latin per ICAO Doc 9303, preserving
// the case of the source letters.
//
// Characters outside the table (Latin, digits, spaces, punctuation) are passed
// through unchanged. The soft sign (ь/Ь) is dropped.
//
// The case of multi-letter sequences is chosen from context:
//
//	"Жук"    -> "Zhuk"   (uppercase before lowercase - Title Case)
//	"ЖУК"    -> "ZHUK"   (uppercase before uppercase - UPPER CASE)
//	"жук"    -> "zhuk"   (lowercase)
func ToLatin(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size

		idx, upper, ok := foldIndex(r)
		if !ok {
			// Character outside the table - copy as is.
			b.WriteRune(r)
			continue
		}

		base := _forward[idx]
		if len(base) == 0 {
			continue // ь is dropped
		}
		if !upper {
			b.WriteString(base)
			continue
		}

		writeUpper(&b, base, nextIsUpper(s[i:]))
	}

	return b.String()
}

// writeUpper writes base with its first letter uppercased; if allUpper is set,
// every letter is uppercased. base consists of ASCII a-z, so changing case is a
// subtraction of 0x20 with no Unicode table lookups and no intermediate
// allocations.
func writeUpper(b *strings.Builder, base string, allUpper bool) {
	b.WriteByte(base[0] - 0x20)

	for i := 1; i < len(base); i++ {
		c := base[i]

		if allUpper {
			c -= 0x20
		}

		b.WriteByte(c)
	}
}

// nextIsUpper reports whether the rest of the string begins with an uppercase
// Russian letter. This distinguishes an all-uppercase word ("ЖУК") from a
// Title-Case word ("Жук").
func nextIsUpper(rest string) bool {
	if len(rest) == 0 {
		return false
	}

	r, _ := utf8.DecodeRuneInString(rest)
	_, upper, ok := foldIndex(r)

	return ok && upper
}

// trieNode is a node of the prefix tree of Latin sequences used for reverse
// transliteration.
type trieNode struct {
	children [26]*trieNode
	cyr      rune
	hasVal   bool
}

// _reverseTrie is a prefix tree (trie) of Latin sequences. Longest-match lookup
// (maximal munch) runs in a single pass over the tree - O(depth), at most 4 -
// with no allocation, instead of linearly scanning the whole mapping table with
// per-character comparisons at every position.
var _reverseTrie = buildReverseTrie()

func buildReverseTrie() *trieNode {
	// Letters that merge into others during forward transliteration (ё, э, й)
	// have no reverse mapping: e->е, i->и. The soft sign is unrecoverable (ie->ъ).
	pairs := []struct {
		latin string
		cyr   rune
	}{
		{"a", 'а'}, {"b", 'б'}, {"v", 'в'}, {"g", 'г'}, {"d", 'д'},
		{"e", 'е'}, {"z", 'з'}, {"i", 'и'}, {"k", 'к'}, {"l", 'л'},
		{"m", 'м'}, {"n", 'н'}, {"o", 'о'}, {"p", 'п'}, {"r", 'р'},
		{"s", 'с'}, {"t", 'т'}, {"u", 'у'}, {"f", 'ф'}, {"y", 'ы'},
		{"zh", 'ж'}, {"kh", 'х'}, {"ts", 'ц'}, {"ch", 'ч'}, {"sh", 'ш'},
		{"shch", 'щ'}, {"ie", 'ъ'}, {"iu", 'ю'}, {"ia", 'я'},
	}
	root := &trieNode{}

	for _, p := range pairs {
		n := root

		for i := 0; i < len(p.latin); i++ {
			k := p.latin[i] - 'a'

			if n.children[k] == nil {
				n.children[k] = &trieNode{}
			}

			n = n.children[k]
		}

		n.cyr = p.cyr

		n.hasVal = true
	}

	return root
}

// match finds the longest Latin sequence starting at position i (case
// insensitive) and returns its Cyrillic letter and the number of bytes consumed
// (0 if there is no match).
func (n *trieNode) match(s string, i int) (cyr rune, matchLen int) {
	for j := i; j < len(s); j++ {
		c := s[j]

		if c >= 'A' && c <= 'Z' {
			c += 0x20 // fold to ASCII lowercase
		}
		if c < 'a' || c > 'z' {
			break
		}

		child := n.children[c-'a']
		if child == nil {
			break
		}

		n = child

		if child.hasVal {
			cyr = child.cyr
			matchLen = j - i + 1
		}
	}

	return cyr, matchLen
}

// ToCyrillic performs best-effort reverse transliteration from Latin to
// Cyrillic.
//
// It is not an exact inverse of [ToLatin]: the ICAO standard loses information
// (е/ё/э->e, и/й->i, ь dropped), so the reverse mapping is ambiguous. For
// example, ToCyrillic("Elena") returns "Елена", but the source could also have
// been "Ёлена".
//
// Case is preserved: the case of the resulting letter is determined by the
// first character of the recognized Latin sequence ("ZH"/"Zh"->Ж, "zh"->ж).
// Unrecognized characters are copied unchanged.
func ToCyrillic(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 2) // ASCII Latin -> 2 bytes per Cyrillic rune

	for i := 0; i < len(s); {
		if cyr, matchLen := _reverseTrie.match(s, i); matchLen > 0 {
			if c := s[i]; c >= 'A' && c <= 'Z' {
				b.WriteRune(unicode.ToUpper(cyr))
			} else {
				b.WriteRune(cyr)
			}

			i += matchLen
			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		b.WriteRune(r)

		i += size
	}

	return b.String()
}
