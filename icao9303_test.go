package icao9303

import (
	"testing"
)

func TestToLatin_Alphabet(t *testing.T) {
	// Full table of lowercase letters per ICAO Doc 9303.
	cases := map[string]string{
		"а": "a", "б": "b", "в": "v", "г": "g", "д": "d",
		"е": "e", "ё": "e", "ж": "zh", "з": "z", "и": "i",
		"й": "i", "к": "k", "л": "l", "м": "m", "н": "n",
		"о": "o", "п": "p", "р": "r", "с": "s", "т": "t",
		"у": "u", "ф": "f", "х": "kh", "ц": "ts", "ч": "ch",
		"ш": "sh", "щ": "shch", "ъ": "ie", "ы": "y", "ь": "",
		"э": "e", "ю": "iu", "я": "ia",
	}

	for in, want := range cases {
		if got := ToLatin(in); got != want {
			t.Errorf("ToLatin(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToLatin_Words(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Иванов", "Ivanov"},
		{"Пётр", "Petr"},
		{"Щёлково", "Shchelkovo"},
		{"Объёмный", "Obieemnyi"}, // ъ->ie, ё->e, й->i
		{"Юлия", "Iuliia"},
		{"Яна", "Iana"},
		{"Ольга", "Olga"},  // ь is dropped
		{"Дарья", "Daria"}, // ь dropped, я->ia; д а р ь я = d a r ia
		{"Цой", "Tsoi"},
		{"Хабаровск", "Khabarovsk"},
		{"Чехов", "Chekhov"},
		{"Жжёнов", "Zhzhenov"},
		{"Съезд", "Sieezd"}, // с ъ е з д = s ie e z d
		{"Эдуард", "Eduard"},
	}

	for _, c := range cases {
		if got := ToLatin(c.in); got != c.want {
			t.Errorf("ToLatin(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToLatin_Case(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"жук", "zhuk"}, // lowercase
		{"Жук", "Zhuk"}, // Title Case
		{"ЖУК", "ZHUK"}, // UPPER CASE
		{"Щи", "Shchi"}, // multi-letter, Title Case
		{"ЩИ", "SHCHI"}, // multi-letter, UPPER CASE
		{"ИВАНОВ", "IVANOV"},
		{"Ъ", "Ie"}, // single uppercase -> Title
		{"Я", "Ia"},
		{"ЯНА", "IANA"},
		{"МХАТ", "MKHAT"}, // х inside an all-uppercase word
	}

	for _, c := range cases {
		if got := ToLatin(c.in); got != c.want {
			t.Errorf("ToLatin(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToLatin_Passthrough(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Дом 12, кв. 3", "Dom 12, kv. 3"},
		{"e-mail: тест@почта", "e-mail: test@pochta"},
		{"ГОСТ 7.79-2000", "GOST 7.79-2000"},
		{"", ""},
		{"ASCII only 123", "ASCII only 123"},
		{"Иван-Пётр", "Ivan-Petr"},
	}

	for _, c := range cases {
		if got := ToLatin(c.in); got != c.want {
			t.Errorf("ToLatin(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToCyrillic_BestEffort(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Ivanov", "Иванов"},
		{"Shchelkovo", "Щелково"},
		{"Iuliia", "Юлия"},
		{"Iana", "Яна"},
		{"Tsoi", "Цои"}, // й is unrecoverable: i->и
		{"Khabarovsk", "Хабаровск"},
		{"Chekhov", "Чехов"},
		{"Sieezd", "Съезд"}, // с+ъ(ie)+е(e)+з+д - no loss here
		{"Dom 12", "Дом 12"},
		{"", ""},
	}

	for _, c := range cases {
		if got := ToCyrillic(c.in); got != c.want {
			t.Errorf("ToCyrillic(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToCyrillic_Case(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"zh", "ж"},
		{"Zh", "Ж"},
		{"ZH", "Ж"},
		{"SHCH", "Щ"},
		{"IVANOV", "ИВАНОВ"},
	}

	for _, c := range cases {
		if got := ToCyrillic(c.in); got != c.want {
			t.Errorf("ToCyrillic(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestRoundTrip_Lossless checks that for words without "lossy" letters
// (ё, э, й, ь, and doublings such as ъе) the reverse transformation
// recovers the original text.
func TestRoundTrip_Lossless(t *testing.T) {
	words := []string{
		"Иванов",
		"Хабаровск",
		"Чехов",
		"Москва",
		"Щука",
		"Юра", // without й (Юрий), so the transformation is reversible
		"Яна",
	}

	for _, w := range words {
		if got := ToCyrillic(ToLatin(w)); got != w {
			t.Errorf("roundtrip(%q) = %q", w, got)
		}
	}
}

func BenchmarkToLatin(b *testing.B) {
	const s = "И когда мне уже показалось, что я завязал… они снова меня туда затащили"

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ToLatin(s)
	}
}

func BenchmarkToCyrillic(b *testing.B) {
	const s = "I kogda mne uzhe pokazalos, chto ia zaviazal… oni snova menia tuda zatashchili"

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ToCyrillic(s)
	}
}
