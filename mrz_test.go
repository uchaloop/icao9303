package icao9303

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestBuildMRZ_ICAOExample checks MRZ generation against the canonical ICAO
// Doc 9303 TD3 example (Anna Maria Eriksson) - the reference for check digits.
func TestBuildMRZ_ICAOExample(t *testing.T) {
	m, err := BuildMRZ(Passport{
		DocumentType:   "P",
		IssuingCountry: "UTO",
		Surname:        "ERIKSSON",
		GivenNames:     "ANNA MARIA",
		Number:         "L898902C3",
		Nationality:    "UTO",
		BirthDate:      "740812",
		Sex:            "F",
		ExpiryDate:     "120415",
		PersonalNumber: "ZE184226B",
	})
	if err != nil {
		t.Fatalf("BuildMRZ returned an error: %v", err)
	}

	wantLine1 := "P<UTOERIKSSON<<ANNA<MARIA<<<<<<<<<<<<<<<<<<<"
	wantLine2 := "L898902C36UTO7408122F1204159ZE184226B<<<<<10"

	if m.Line1 != wantLine1 {
		t.Errorf("Line1:\n got %q\nwant %q", m.Line1, wantLine1)
	}
	if m.Line2 != wantLine2 {
		t.Errorf("Line2:\n got %q\nwant %q", m.Line2, wantLine2)
	}
	if len(m.Line1) != MRZLineLength || len(m.Line2) != MRZLineLength {
		t.Errorf("line lengths: %d and %d, want %d each", len(m.Line1), len(m.Line2), MRZLineLength)
	}
}

// TestBuildMRZ_Cyrillic checks that a Cyrillic surname and given names are
// transliterated automatically per ICAO Doc 9303.
func TestBuildMRZ_Cyrillic(t *testing.T) {
	m, err := BuildMRZ(Passport{
		IssuingCountry: "RUS",
		Surname:        "Иванов",
		GivenNames:     "Анна Мария",
		Number:         "760123456",
		Nationality:    "RUS",
		BirthDate:      "900201",
		Sex:            "F",
		ExpiryDate:     "300201",
	})
	if err != nil {
		t.Fatalf("BuildMRZ returned an error: %v", err)
	}

	// P (default) + < + RUS + IVANOV<<ANNA<MARIIA...
	const wantPrefix = "P<RUSIVANOV<<ANNA<MARIIA"
	if !strings.HasPrefix(m.Line1, wantPrefix) {
		t.Errorf("Line1 = %q, want prefix %q", m.Line1, wantPrefix)
	}
	if len(m.Line1) != MRZLineLength {
		t.Errorf("len(Line1) = %d, want %d", len(m.Line1), MRZLineLength)
	}
	// No personal number - the personal-number check digit is "<".
	if got := m.Line2[42]; got != '<' {
		t.Errorf("personal-number check digit (pos. 43) = %q, want '<'", string(got))
	}
}

// TestBuildMRZ_NoPersonalNumber checks that with an empty personal number the
// field is filled with "<" and its check digit is "<".
func TestBuildMRZ_NoPersonalNumber(t *testing.T) {
	m, err := BuildMRZ(Passport{
		IssuingCountry: "RUS",
		Surname:        "PETROV",
		GivenNames:     "IVAN",
		Number:         "123456789",
		Nationality:    "RUS",
		BirthDate:      "850615",
		Sex:            "M",
		ExpiryDate:     "350615",
	})
	if err != nil {
		t.Fatalf("BuildMRZ returned an error: %v", err)
	}
	personalField := m.Line2[28:42] // pos. 29-42
	if personalField != strings.Repeat("<", 14) {
		t.Errorf("personal number = %q, want 14 '<' characters", personalField)
	}
	if m.Line2[42] != '<' { // pos. 43
		t.Errorf("personal-number check digit = %q, want '<'", string(m.Line2[42]))
	}
}

// TestBuildMRZ_NameFormatting checks clause 12.1.1 rules: double surnames are
// separated by "<", apostrophes are removed, surname and name by "<<".
func TestBuildMRZ_NameFormatting(t *testing.T) {
	m, err := BuildMRZ(Passport{
		IssuingCountry: "RUS",
		Surname:        "Петров-Водкин", // double surname
		GivenNames:     "Кузьма",
		Number:         "123456789",
		Nationality:    "RUS",
		BirthDate:      "800101",
		Sex:            "M",
		ExpiryDate:     "300101",
	})
	if err != nil {
		t.Fatalf("BuildMRZ returned an error: %v", err)
	}
	const wantPrefix = "P<RUSPETROV<VODKIN<<KUZMA"
	if !strings.HasPrefix(m.Line1, wantPrefix) {
		t.Errorf("Line1 = %q, want prefix %q", m.Line1, wantPrefix)
	}
}

func TestNameFold(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"O'Brien", "OBRIEN"},              // apostrophe removed
		{"O’Brien", "OBRIEN"},              // typographic apostrophe
		{"PETROV-VODKIN", "PETROV<VODKIN"}, // hyphen -> "<"
		{"Anna Maria", "ANNA<MARIA"},       // space -> "<"
		{"", ""},
	}
	for _, c := range cases {
		if got := nameFold(c.in); got != c.want {
			t.Errorf("nameFold(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCheckDigit(t *testing.T) {
	cases := []struct {
		in   string
		want byte
	}{
		{"L898902C3", '6'},
		{"740812", '2'},
		{"120415", '9'},
		{"ZE184226B<<<<<", '1'},
		{"", '0'},
		{"<<<<<", '0'},
	}
	for _, c := range cases {
		if got := checkDigit(c.in); got != c.want {
			t.Errorf("checkDigit(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCharValue(t *testing.T) {
	cases := map[byte]int{'0': 0, '9': 9, 'A': 10, 'Z': 35, '<': 0, ' ': 0}
	for c, want := range cases {
		if got := charValue(c); got != want {
			t.Errorf("charValue(%q) = %d, want %d", c, got, want)
		}
	}
}

func TestBuildMRZ_Errors(t *testing.T) {
	valid := Passport{
		IssuingCountry: "RUS", Surname: "PETROV", GivenNames: "IVAN",
		Number: "123456789", Nationality: "RUS",
		BirthDate: "850615", Sex: "M", ExpiryDate: "350615",
	}

	cases := []struct {
		name  string
		patch func(*Passport)
	}{
		{"empty surname", func(p *Passport) { p.Surname = "" }},
		{"short issuing country", func(p *Passport) { p.IssuingCountry = "RU" }},
		{"short nationality", func(p *Passport) { p.Nationality = "RU" }},
		{"long number", func(p *Passport) { p.Number = "1234567890" }},
		{"bad birth date", func(p *Passport) { p.BirthDate = "85061" }},
		{"non-digit date", func(p *Passport) { p.ExpiryDate = "35O615" }},
		{"invalid sex", func(p *Passport) { p.Sex = "X" }},
		{"long personal number", func(p *Passport) { p.PersonalNumber = strings.Repeat("A", 15) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := valid
			c.patch(&p)
			if _, err := BuildMRZ(p); err == nil {
				t.Errorf("expected an error for case %q, got nil", c.name)
			}
		})
	}
}

func TestMRZ_String(t *testing.T) {
	m := MRZ{Line1: "A", Line2: "B"}
	if got := m.String(); got != "A\nB" {
		t.Errorf("String() = %q, want %q", got, "A\nB")
	}
}

func BenchmarkBuildMRZ(b *testing.B) {
	p := Passport{
		IssuingCountry: "RUS", Surname: "Петров-Водкин", GivenNames: "Кузьма",
		Number: "760123456", Nationality: "RUS",
		BirthDate: "800101", Sex: "M", ExpiryDate: "300101", PersonalNumber: "ABC123",
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = BuildMRZ(p)
	}
}

func BenchmarkParseMRZ(b *testing.B) {
	l1 := "P<UTOERIKSSON<<ANNA<MARIA<<<<<<<<<<<<<<<<<<<"
	l2 := "L898902C36UTO7408122F1204159ZE184226B<<<<<10"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseMRZ(l1, l2)
	}
}

// TestParseMRZ_ICAOExample parses the canonical TD3 example and checks that all
// five check digits match and the fields are extracted correctly.
func TestParseMRZ_ICAOExample(t *testing.T) {
	line1 := "P<UTOERIKSSON<<ANNA<MARIA<<<<<<<<<<<<<<<<<<<"
	line2 := "L898902C36UTO7408122F1204159ZE184226B<<<<<10"

	p, err := ParseMRZ(line1, line2)
	if err != nil {
		t.Fatalf("ParseMRZ returned an error: %v", err)
	}
	if !p.Valid {
		t.Error("Valid = false, want true")
	}
	want := ParsedMRZ{
		DocumentType:   "P",
		IssuingCountry: "UTO",
		Surname:        "ERIKSSON",
		GivenNames:     []string{"ANNA", "MARIA"},
		Number:         "L898902C3",
		Nationality:    "UTO",
		BirthDate:      "740812",
		Sex:            "F",
		ExpiryDate:     "120415",
		PersonalNumber: "ZE184226B",
		Valid:          true,
	}
	if !reflect.DeepEqual(p, want) {
		t.Errorf("ParseMRZ:\n got %+v\nwant %+v", p, want)
	}
}

// TestParseMRZ_RoundTrip checks that BuildMRZ and ParseMRZ are consistent.
func TestParseMRZ_RoundTrip(t *testing.T) {
	src := Passport{
		IssuingCountry: "RUS",
		Surname:        "Петров-Водкин",
		GivenNames:     "Кузьма",
		Number:         "760123456",
		Nationality:    "RUS",
		BirthDate:      "800101",
		Sex:            "M",
		ExpiryDate:     "300101",
		PersonalNumber: "ABC123",
	}
	m, err := BuildMRZ(src)
	if err != nil {
		t.Fatalf("BuildMRZ: %v", err)
	}
	p, err := ParseMRZ(m.Line1, m.Line2)
	if err != nil {
		t.Fatalf("ParseMRZ returned an error: %v", err)
	}
	if !p.Valid {
		t.Error("Valid = false, want true")
	}
	if p.Surname != "PETROV VODKIN" {
		t.Errorf("Surname = %q, want %q", p.Surname, "PETROV VODKIN")
	}
	if len(p.GivenNames) != 1 || p.GivenNames[0] != "KUZMA" {
		t.Errorf("GivenNames = %v, want [KUZMA]", p.GivenNames)
	}
	if p.Number != "760123456" || p.PersonalNumber != "ABC123" {
		t.Errorf("Number=%q PersonalNumber=%q", p.Number, p.PersonalNumber)
	}
}

// TestParseMRZ_BadCheckDigits checks that corruption yields Valid=false and the
// specific check-digit error.
func TestParseMRZ_BadCheckDigits(t *testing.T) {
	line1 := "P<UTOERIKSSON<<ANNA<MARIA<<<<<<<<<<<<<<<<<<<"
	// Corrupt the document-number check digit (pos. 10): 6 -> 5.
	line2 := "L898902C35UTO7408122F1204159ZE184226B<<<<<10"

	p, err := ParseMRZ(line1, line2)
	if err == nil {
		t.Fatal("expected a check-digit error, got nil")
	}
	if p.Valid {
		t.Error("Valid = true, want false")
	}
	if !errors.Is(err, ErrNumberCheckDigit) {
		t.Errorf("error = %v, want ErrNumberCheckDigit", err)
	}
	// The composite check digit is broken as well.
	if !errors.Is(err, ErrCompositeCheckDigit) {
		t.Errorf("want ErrCompositeCheckDigit too, got %v", err)
	}
	// The data must still be extracted.
	if p.Surname != "ERIKSSON" {
		t.Errorf("Surname = %q, want ERIKSSON", p.Surname)
	}
}

func TestParseMRZ_BadLength(t *testing.T) {
	if _, err := ParseMRZ("too short", "also short"); err == nil {
		t.Error("expected a length error, got nil")
	}
}

// TestParseMRZ_NoPersonalNumber checks the empty personal-number case: the "<"
// check digit is accepted as valid.
func TestParseMRZ_NoPersonalNumber(t *testing.T) {
	m, err := BuildMRZ(Passport{
		IssuingCountry: "RUS", Surname: "IVANOV", GivenNames: "IVAN",
		Number: "123456789", Nationality: "RUS",
		BirthDate: "850615", Sex: "M", ExpiryDate: "350615",
	})
	if err != nil {
		t.Fatalf("BuildMRZ: %v", err)
	}

	p, err := ParseMRZ(m.Line1, m.Line2)
	if err != nil {
		t.Fatalf("ParseMRZ: %v", err)
	}
	if !p.Valid {
		t.Error("Valid = false, want true")
	}
	if len(p.PersonalNumber) != 0 {
		t.Errorf("PersonalNumber = %q, want empty", p.PersonalNumber)
	}
}
