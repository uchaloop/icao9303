package icao9303

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// MRZ and the related functions build the Machine-Readable Zone (MRZ) of a TD3
// passport (two lines of 44 characters) per ICAO Doc 9303 and clauses 12.1.1,
// 12.1.2, 12.3.1 and 12.3.2 of Order No. 2113 of the Ministry of Foreign
// Affairs of the Russian Federation, dated 12 February 2020.
//
// First-line layout (clause 12.1.1):
//
//	pos. 1      document type ("P")
//	pos. 2      document subtype ("<" if unused)
//	pos. 3-5    issuing state code (3 letters)
//	pos. 6-44   surname and given names: SURNAME<<GIVEN<NAMES, padded with "<"
//
// Second-line layout (clause 12.1.2) and check digits (clause 12.3.1):
//
//	pos. 1-9    document number
//	pos. 10     document-number check digit             <- CD
//	pos. 11-13  nationality (3 letters)
//	pos. 14-19  date of birth (YYMMDD)
//	pos. 20     date-of-birth check digit               <- CD
//	pos. 21     sex ("M", "F" or "<")
//	pos. 22-27  date of expiry (YYMMDD)
//	pos. 28     date-of-expiry check digit              <- CD
//	pos. 29-42  personal number (or "<")
//	pos. 43     personal-number check digit             <- CD
//	pos. 44     composite check digit                   <- CD

// MRZLineLength is the length of a single TD3 MRZ line.
const MRZLineLength = 44

// _filler is the MRZ filler character ("<").
const _filler = '<'

// MRZ holds the two generated Machine-Readable Zone lines.
type MRZ struct {
	Line1 string
	Line2 string
}

// String returns both MRZ lines separated by a newline.
func (m MRZ) String() string {
	return m.Line1 + "\n" + m.Line2
}

// Passport holds the data used to build a TD3 passport MRZ.
//
// Surname and GivenNames may be given in Cyrillic or Latin - they are
// transliterated with [ToLatin] and normalized automatically. The remaining
// fields must be supplied in their final form.
type Passport struct {
	DocumentType   string // document type, usually "P"; defaults to "P"
	IssuingCountry string // issuing state code (3 letters), e.g. "RUS"
	Surname        string // surname (Cyrillic or Latin)
	GivenNames     string // given names separated by spaces (Cyrillic or Latin)
	Number         string // document number (up to 9 characters)
	Nationality    string // nationality (3 letters), e.g. "RUS"
	BirthDate      string // date of birth "YYMMDD"
	Sex            string // "M", "F" or "" (=> "<")
	ExpiryDate     string // date of expiry "YYMMDD"
	PersonalNumber string // personal number (up to 14 characters), may be empty
}

// BuildMRZ builds a TD3 MRZ from passport data, computing all five check digits
// (positions 10, 20, 28, 43, 44).
//
// It returns an error if a required field is missing or exceeds its allowed
// length, or if a date is not exactly six digits.
func BuildMRZ(p Passport) (MRZ, error) {
	docType := p.DocumentType
	if len(docType) == 0 {
		docType = "P"
	}

	if err := validate(p, docType); err != nil {
		return MRZ{}, err
	}

	line1 := buildLine1(docType, p.IssuingCountry, p.Surname, p.GivenNames)

	number := padField(mrzFold(p.Number), 9)
	numberCD := checkDigit(number)

	birth := mrzFold(p.BirthDate)
	birthCD := checkDigit(birth)

	sexByte := byte(_filler)
	if len(p.Sex) != 0 {
		sexByte = p.Sex[0] // the validator guarantees "M"/"F"
	}

	expiry := mrzFold(p.ExpiryDate)
	expiryCD := checkDigit(expiry)

	personal := padField(mrzFold(p.PersonalNumber), 14)
	// Per ICAO the personal-number check digit is computed over 14 characters;
	// if the field is unused the check digit is "<".
	personalCD := byte(_filler)
	if len(strings.Trim(personal, string(_filler))) != 0 {
		personalCD = checkDigit(personal)
	}

	// The composite check digit (pos. 44) is computed over the combined sequence
	// of positions 1-10, 14-20, 22-43 (number+CD, birth+CD, expiry+CD, code+CD).
	compositeCD := compositeCheckDigit(number, numberCD, birth, birthCD,
		expiry, expiryCD, personal, personalCD)

	// Assemble the second line without intermediate string concatenations.
	var b strings.Builder
	b.Grow(MRZLineLength)
	b.WriteString(number)                 // pos. 1-9
	b.WriteByte(numberCD)                 // pos. 10
	b.WriteString(mrzFold(p.Nationality)) // pos. 11-13
	b.WriteString(birth)                  // pos. 14-19
	b.WriteByte(birthCD)                  // pos. 20
	b.WriteByte(sexByte)                  // pos. 21
	b.WriteString(expiry)                 // pos. 22-27
	b.WriteByte(expiryCD)                 // pos. 28
	b.WriteString(personal)               // pos. 29-42
	b.WriteByte(personalCD)               // pos. 43
	b.WriteByte(compositeCD)              // pos. 44

	return MRZ{Line1: line1, Line2: b.String()}, nil
}

func validate(p Passport, docType string) error {
	switch {
	case len(mrzFold(docType)) > 2:
		return fmt.Errorf("document type %q is longer than 2 characters", docType)
	case len(mrzFold(p.IssuingCountry)) != 3:
		return fmt.Errorf("issuing state code %q must be 3 characters", p.IssuingCountry)
	case len(mrzFold(p.Nationality)) != 3:
		return fmt.Errorf("nationality %q must be 3 characters", p.Nationality)
	case len(strings.TrimSpace(p.Surname)) == 0:
		return errors.New("surname is required")
	case len(mrzFold(p.Number)) > 9:
		return fmt.Errorf("document number %q is longer than 9 characters", p.Number)
	case len(mrzFold(p.PersonalNumber)) > 14:
		return fmt.Errorf("personal number %q is longer than 14 characters", p.PersonalNumber)
	}

	if err := checkDate(p.BirthDate, "birth date"); err != nil {
		return err
	}
	if err := checkDate(p.ExpiryDate, "expiry date"); err != nil {
		return err
	}
	if s := p.Sex; len(s) != 0 && s != "M" && s != "F" {
		return fmt.Errorf("sex %q must be \"M\", \"F\" or empty", s)
	}

	return nil
}

func checkDate(s, name string) error {
	if len(s) != 6 {
		return fmt.Errorf("%s %q must be in YYMMDD format (6 digits)", name, s)
	}

	for _, r := range s {
		if !unicode.IsDigit(r) {
			return fmt.Errorf("%s %q must contain digits only", name, s)
		}
	}

	return nil
}

// buildLine1 builds the first MRZ line (clause 12.1.1).
func buildLine1(docType, country, surname, givenNames string) string {
	head := padField(mrzFold(docType), 2) + padField(mrzFold(country), 3)

	// Name: SURNAME<<GIVEN<NAMES. Name components are separated by a single "<".
	name := nameFold(ToLatin(surname))
	if given := givenParts(givenNames); len(given) != 0 {
		name += string(_filler) + string(_filler) + given
	}

	nameField := padField(name, MRZLineLength-len(head))
	// Truncate if the name does not fit (per ICAO - truncated from the end).
	if len(nameField) > MRZLineLength-len(head) {
		nameField = nameField[:MRZLineLength-len(head)]
	}
	return head + nameField
}

// givenParts transliterates given names and joins them with a single "<".
func givenParts(givenNames string) string {
	fields := strings.Fields(givenNames)
	parts := make([]string, 0, len(fields))

	for _, f := range fields {
		if folded := nameFold(ToLatin(f)); len(folded) != 0 {
			parts = append(parts, folded)
		}
	}

	return strings.Join(parts, string(_filler))
}

// nameFold folds a surname or given name into the MRZ character set per clause
// 12.1.1: Latin letters are uppercased, apostrophes are removed, and spaces,
// hyphens and other separators become "<". Diacritics are not expected because
// the names have already passed through ToLatin.
func nameFold(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		switch {
		case r == '\'' || r == '’' || r == '`' || r == 'ʼ':
			// "apostrophes are removed" (clause 12.1.1) - the character is skipped.
			continue
		case r >= 'a' && r <= 'z':
			b.WriteRune(unicode.ToUpper(r))
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune(_filler)
		}
	}

	return b.String()
}

// mrzFold folds a string into the MRZ character set: Latin letters are
// uppercased, digits are kept, and spaces and any other characters become "<".
// Diacritics are not expected because names have already passed through ToLatin.
func mrzFold(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(unicode.ToUpper(r))
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune(_filler)
		}
	}

	return b.String()
}

// padField pads s on the right with "<" up to length n.
func padField(s string, n int) string {
	if len(s) >= n {
		return s
	}

	return s + strings.Repeat(string(_filler), n-len(s))
}

// _checkDigitWeights is the continuously repeating 7-3-1 weighting function
// (clause 12.3.2).
var _checkDigitWeights = [3]int{7, 3, 1}

// checkDigit computes a modulo-10 check digit (weights 7-3-1, clause 12.3.2):
// digits count as their value, letters A-Z as 10..35, and "<" as 0. It returns
// a digit character ('0'..'9'). The compiler turns division by 10 into a
// multiply-and-shift; a manual shift is inappropriate here (10 is not a power
// of two).
func checkDigit(s string) byte {
	sum, weight := 0, 0

	for i := 0; i < len(s); i++ {
		sum += charValue(s[i]) * _checkDigitWeights[weight]
		weight++

		if weight == 3 {
			weight = 0
		}
	}

	return byte('0' + sum%10)
}

// compositeCheckDigit computes the composite check digit (pos. 44) over the
// combined sequence (positions 1-10, 14-20, 22-43) without intermediate strings:
// the weight index runs continuously across all parts.
func compositeCheckDigit(
	number string,
	numberCD byte,
	birth string,
	birthCD byte,
	expiry string,
	expiryCD byte,
	personal string,
	personalCD byte,
) byte {
	sum, weight := 0, 0
	step := func(value int) {
		sum += value * _checkDigitWeights[weight]
		weight++

		if weight == 3 {
			weight = 0
		}
	}
	add := func(s string) {
		for i := 0; i < len(s); i++ {
			step(charValue(s[i]))
		}
	}
	addByte := func(c byte) {
		step(charValue(c))
	}
	add(number)
	addByte(numberCD)
	add(birth)
	addByte(birthCD)
	add(expiry)
	addByte(expiryCD)
	add(personal)
	addByte(personalCD)

	return byte('0' + sum%10)
}

// charValue returns the numeric value of an MRZ character for the check digit.
func charValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	default: // "<" and everything else
		return 0
	}
}

// ParsedMRZ holds the data parsed from a TD3 passport Machine-Readable Zone.
// Number and PersonalNumber are returned without trailing "<" fillers; in the
// surname and given names the "<" separators are converted back to spaces.
type ParsedMRZ struct {
	DocumentType   string   // document type (pos. 1-2), e.g. "P"
	IssuingCountry string   // issuing state code (pos. 3-5)
	Surname        string   // surname (double surnames joined by a space)
	GivenNames     []string // given names
	Number         string   // document number
	Nationality    string   // nationality
	BirthDate      string   // date of birth "YYMMDD"
	Sex            string   // "M", "F" or "" (for "<")
	ExpiryDate     string   // date of expiry "YYMMDD"
	PersonalNumber string   // personal number (empty if not set)
	Valid          bool     // true if all five check digits matched
}

// Errors returned by ParseMRZ when a check digit does not match.
var (
	ErrNumberCheckDigit    = errors.New("invalid document-number check digit (pos. 10)")
	ErrBirthCheckDigit     = errors.New("invalid date-of-birth check digit (pos. 20)")
	ErrExpiryCheckDigit    = errors.New("invalid date-of-expiry check digit (pos. 28)")
	ErrPersonalCheckDigit  = errors.New("invalid personal-number check digit (pos. 43)")
	ErrCompositeCheckDigit = errors.New("invalid composite check digit (pos. 44)")
)

// ParseMRZ parses the two lines of a TD3 passport Machine-Readable Zone and
// verifies all five check digits (positions 10, 20, 28, 43, 44) per clauses
// 12.3.1-12.3.2.
//
// Trailing newlines and spaces are ignored; each line must contain exactly 44
// characters. On a structural error (wrong length) it returns a zero ParsedMRZ
// and an error.
//
// If the structure is valid but one or more check digits do not match, it
// returns a populated ParsedMRZ with Valid=false and a joined error (see
// errors.Join and the Err*CheckDigit sentinels) so the caller can both read the
// data and learn which checks failed.
func ParseMRZ(line1, line2 string) (ParsedMRZ, error) {
	l1 := strings.TrimRight(line1, "\r\n ")
	l2 := strings.TrimRight(line2, "\r\n ")

	if len(l1) != MRZLineLength || len(l2) != MRZLineLength {
		return ParsedMRZ{}, fmt.Errorf(
			"MRZ lines must be %d characters each, got %d and %d",
			MRZLineLength, len(l1), len(l2))
	}

	surname, given := parseName(l1[5:44])
	p := ParsedMRZ{
		DocumentType:   trimFiller(l1[0:2]),
		IssuingCountry: l1[2:5],
		Surname:        surname,
		GivenNames:     given,
		Number:         trimFiller(l2[0:9]),
		Nationality:    l2[10:13],
		BirthDate:      l2[13:19],
		Sex:            parseSex(l2[20]),
		ExpiryDate:     l2[21:27],
		PersonalNumber: trimFiller(l2[28:42]),
	}

	// Verify the five check digits.
	var errs []error
	if checkDigit(l2[0:9]) != l2[9] {
		errs = append(errs, ErrNumberCheckDigit)
	}
	if checkDigit(l2[13:19]) != l2[19] {
		errs = append(errs, ErrBirthCheckDigit)
	}
	if checkDigit(l2[21:27]) != l2[27] {
		errs = append(errs, ErrExpiryCheckDigit)
	}
	if !personalCheckDigitOK(l2[28:42], l2[42]) {
		errs = append(errs, ErrPersonalCheckDigit)
	}
	composite := compositeCheckDigit(l2[0:9], l2[9], l2[13:19], l2[19],
		l2[21:27], l2[27], l2[28:42], l2[42])
	if composite != l2[43] {
		errs = append(errs, ErrCompositeCheckDigit)
	}

	if len(errs) > 0 {
		return p, errors.Join(errs...)
	}
	p.Valid = true

	return p, nil
}

// personalCheckDigitOK verifies the personal-number check digit (pos. 43). Per
// clause 12.3.2, when there is no personal number both "<" and "0" are allowed.
func personalCheckDigitOK(field string, cd byte) bool {
	if len(strings.Trim(field, string(_filler))) == 0 {
		return cd == _filler || cd == '0'
	}

	return checkDigit(field) == cd
}

// parseName splits the MRZ name field (SURNAME<<GIVEN<NAMES) into surname and
// given names.
func parseName(field string) (surname string, given []string) {
	field = strings.TrimRight(field, string(_filler))
	parts := strings.SplitN(field, string(_filler)+string(_filler), 2)
	surname = strings.ReplaceAll(parts[0], string(_filler), " ")

	if len(parts) == 2 {
		for _, g := range strings.Split(parts[1], string(_filler)) {
			if len(g) != 0 {
				given = append(given, g)
			}
		}
	}

	return surname, given
}

// parseSex converts the MRZ sex character: "<" is treated as "unspecified" ("").
func parseSex(c byte) string {
	if c == _filler {
		return ""
	}

	return string(c)
}

// trimFiller removes trailing "<" filler characters.
func trimFiller(s string) string {
	return strings.TrimRight(s, string(_filler))
}
