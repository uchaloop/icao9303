# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

- No changes yet.

## [1.0.1] - 2026-07-28

### Added

- Contribution guide (`CONTRIBUTING.md`).
- CI workflow (GitHub Actions: gofmt, go vet, go test) and a CI status badge.

## [1.0.0] - 2026-07-28

Initial release. Cyrillic transliteration and passport MRZ (TD3) support based
on the official standards: [ICAO Doc 9303, Part 3](https://www.icao.int/publications/doc-series/doc-9303)
(transliteration table and §4.9 check-digit algorithm) and
[Order No. 2113 of the Russian Ministry of Foreign Affairs of 12.02.2020](http://publication.pravo.gov.ru/Document/View/0001202008250067)
(clauses 12.1.1, 12.1.2, 12.3.1, 12.3.2).

### Added

- `ToLatin` - Cyrillic -> Latin transliteration per the official ICAO Doc 9303
  table, preserving letter case (including multi-letter sequences such as
  `Жук`->`Zhuk` and `ЖУК`->`ZHUK`). Characters outside the table pass through
  unchanged; the soft sign `ь` is omitted.
- `ToCyrillic` - best-effort reverse transliteration (maximal-munch matching).
  Not an exact inverse due to the lossy standard (е/ё/э->e, и/й->i).
- `BuildMRZ` - builds the two 44-character TD3 MRZ lines from `Passport` data,
  computing all five check digits (positions 10, 20, 28, 43, 44) with the
  `731 731`, modulo-10 algorithm. Names are transliterated automatically;
  apostrophes are dropped and double surnames are separated by `<` per 12.1.1.
- `ParseMRZ` - parses two MRZ lines into `ParsedMRZ` and verifies all five
  check digits. Returns populated data with `Valid == false` and a joined error
  on check-digit mismatch; sentinel errors `ErrNumberCheckDigit`,
  `ErrBirthCheckDigit`, `ErrExpiryCheckDigit`, `ErrPersonalCheckDigit`,
  `ErrCompositeCheckDigit` for use with `errors.Is`.
- Types `Passport`, `MRZ`, `ParsedMRZ` and constant `MRZLineLength`.
- Test suite (~95% coverage), including validation against the canonical ICAO
  Doc 9303 example (Anna Maria Eriksson).
- English and Russian READMEs.

### Known limitations

- Partial dates using `<` (clause 12.1.2) are not supported in `BuildMRZ`.
- Long-name truncation is a plain cut to 39 characters, not the ICAO
  component-wise algorithm.
- The sex field accepts only Latin `"M"`/`"F"`.

[Unreleased]: https://github.com/uchaloop/icao9303/compare/v1.0.1...HEAD
[1.0.1]: https://github.com/uchaloop/icao9303/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/uchaloop/icao9303/releases/tag/v1.0.0
