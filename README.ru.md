# icao9303

[![CI](https://github.com/uchaloop/icao9303/actions/workflows/ci.yml/badge.svg)](https://github.com/uchaloop/icao9303/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/uchaloop/icao9303.svg)](https://pkg.go.dev/github.com/uchaloop/icao9303)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

[English](README.md) · **Русский**

Транслитерация кириллица <-> латиница и формирование/разбор машиносчитываемой
зоны (МЧЗ, TD3) паспорта по стандарту **ИКАО Doc 9303**.
Требуется **Go 1.20+**, без зависимостей.

## Источники

Таблица транслитерации, структура МЧЗ и алгоритм контрольной цифры взяты из
официальных стандартов:

- **ИКАО Doc 9303, Part 3 - Specifications Common to all MRTDs** - рекомендуемая
  таблица транслитерации кириллицы и алгоритм контрольной цифры МЧЗ
  (§4.9: модуль 10, веса 7-3-1, `<` = 0, `A`..`Z` = 10..35):
  <https://www.icao.int/publications/doc-series/doc-9303>
- **Приказ МИД России от 12.02.2020 № 2113** (официальное опубликование;
  зарегистрирован в Минюсте 25.08.2020, рег. № 59444) - вводит таблицу ИКАО для
  российских загранпаспортов и задаёт структуру МЧЗ (пп. 12.1.1, 12.1.2, 12.3.1,
  12.3.2):
  <http://publication.pravo.gov.ru/Document/View/0001202008250067>

Библиотека реализует **русское** подмножество таблицы ИКАО. По ИКАО Doc 9303
мягкий знак `ь` не имеет латинского соответствия и опускается.

## Установка

```bash
go get github.com/uchaloop/icao9303
```

## Использование

### Транслитерация

```go
icao9303.ToLatin("Щёлково")   // "Shchelkovo"
icao9303.ToLatin("Юлия")      // "Iuliia"    (регистр сохраняется)
icao9303.ToCyrillic("Ivanov") // "Иванов"    (best-effort; стандарт с потерями)
```

### Формирование МЧЗ

```go
mrz, err := icao9303.BuildMRZ(icao9303.Passport{
	IssuingCountry: "RUS",
	Surname:        "Иванов",     // кириллица или латиница - транслитерируется автоматически
	GivenNames:     "Анна Мария",
	Number:         "760123456",
	Nationality:    "RUS",
	BirthDate:      "900201",     // ГГММДД
	Sex:            "F",
	ExpiryDate:     "300201",
})
fmt.Println(mrz.Line1, mrz.Line2)
```

### Разбор и проверка МЧЗ

```go
p, err := icao9303.ParseMRZ(line1, line2)
// err == nil && p.Valid  -> все пять контрольных цифр сошлись
// при несовпадении p всё равно заполнен, а err сообщает, какая проверка не прошла
// (errors.Is с ErrNumberCheckDigit, ErrBirthCheckDigit, ...)
```

## Лицензия

[MIT](LICENSE).
