# icao9303

[![CI](https://github.com/uchaloop/icao9303/actions/workflows/ci.yml/badge.svg)](https://github.com/uchaloop/icao9303/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/uchaloop/icao9303.svg)](https://pkg.go.dev/github.com/uchaloop/icao9303)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**English** · [Русский](README.ru.md)

Cyrillic <-> Latin transliteration and building/parsing of the passport
Machine-Readable Zone (MRZ, TD3), following the **ICAO Doc 9303** standard.
Requires **Go 1.26+**, no dependencies.

## Sources

The transliteration table, the MRZ layout and the check-digit algorithm are
taken from the official standards:

- **ICAO Doc 9303, Part 3 - Specifications Common to all MRTDs** - the
  recommended Cyrillic transliteration table and the MRZ check-digit algorithm
  (§4.9: modulo 10, weighting 7-3-1, `<` = 0, `A`..`Z` = 10..35):
  <https://www.icao.int/publications/doc-series/doc-9303>
- **Order of the Russian Ministry of Foreign Affairs No. 2113 of 12.02.2020**
  (official publication; Ministry of Justice reg. No. 59444 of 25.08.2020) - it
  adopts the ICAO table for Russian passports and defines the MRZ layout
  (clauses 12.1.1, 12.1.2, 12.3.1, 12.3.2):
  <http://publication.pravo.gov.ru/Document/View/0001202008250067>

The library implements the **Russian** subset of the ICAO table. Per ICAO
Doc 9303 the soft sign `ь` has no Latin equivalent and is dropped.

## Install

```bash
go get github.com/uchaloop/icao9303
```

## Usage

### Transliteration

```go
icao9303.ToLatin("Щёлково")   // "Shchelkovo"
icao9303.ToLatin("Юлия")      // "Iuliia"    (case is preserved)
icao9303.ToCyrillic("Ivanov") // "Иванов"    (best-effort; the standard is lossy)
```

### Build an MRZ

```go
mrz, err := icao9303.BuildMRZ(icao9303.Passport{
	IssuingCountry: "RUS",
	Surname:        "Иванов",     // Cyrillic or Latin - transliterated automatically
	GivenNames:     "Анна Мария",
	Number:         "760123456",
	Nationality:    "RUS",
	BirthDate:      "900201",     // YYMMDD
	Sex:            "F",
	ExpiryDate:     "300201",
})
fmt.Println(mrz.Line1, mrz.Line2)
```

### Parse & verify an MRZ

```go
p, err := icao9303.ParseMRZ(line1, line2)
// err == nil && p.Valid  -> all five check digits matched
// on a mismatch, p is still populated and err reports which check failed
// (errors.Is with ErrNumberCheckDigit, ErrBirthCheckDigit, ...)
```

## License

[MIT](LICENSE).
