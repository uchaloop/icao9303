# Contributing

Thanks for your interest in improving `icao9303`. This is a small, dependency-free
library, so the guidelines are short.

## Prerequisites

- **Go 1.26+**
- No third-party dependencies are allowed - the library must stay
  standard-library only.

## Getting started

Fork the repository, then clone your fork and add the upstream remote:

```bash
git clone https://github.com/<your-username>/icao9303.git
cd icao9303
git remote add upstream https://github.com/uchaloop/icao9303.git
git fetch upstream
```

Or with the [GitHub CLI](https://cli.github.com/):

```bash
gh repo fork uchaloop/icao9303 --clone
```

There is nothing else to install - the library is standard-library only.

## Development workflow

1. Create a branch off `main`.
2. Make your change with tests.
3. Make sure everything passes (see below).
4. Push the branch to your fork and open a pull request describing what changed
   and why.

## Before you push

Run all of the following from the repository root; all must pass:

```bash
gofmt -l .          # must print nothing
go vet ./...
go test ./...
```

If you have [revive](https://github.com/mgechev/revive) installed, run it too:

```bash
revive ./...
```

## Code style

- Format with `gofmt`; follow the
  [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md),
  including the `_` prefix for unexported package-level vars and consts.
- Comments and identifiers are in **English**.
- Keep source punctuation **ASCII**: use `->`, `-`, `...` instead of Unicode
  arrows, em/en dashes, and the ellipsis character. Cyrillic is allowed only
  where it is data (the transliteration table, test inputs) or an official
  document title.
- Error messages state the problem only; do **not** prefix them with the package
  name, and start them lowercase (Go convention).

## Correctness

The transliteration table, the MRZ layout and the check-digit algorithm must
match the official sources:

- [ICAO Doc 9303, Part 3](https://www.icao.int/publications/doc-series/doc-9303)
- [Order No. 2113 of the Russian Ministry of Foreign Affairs of 12.02.2020](http://publication.pravo.gov.ru/Document/View/0001202008250067)

Any change to that logic must cite the relevant clause and keep the tests
(including the canonical ICAO example) green.

## License

By contributing, you agree that your contributions are licensed under the
[MIT License](LICENSE) of this project.
