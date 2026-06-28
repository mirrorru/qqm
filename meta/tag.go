// Created at 2026-06-28
package meta

import "strings"

const (
	// tagCol — имя колонки в БД. Пример: `qqm:"col=user_name"`
	tagCol = "col="

	// tagPK — поле является частью первичного ключа.
	// Пример: `qqm:"pk"`. Порядок определяется порядком объявления в структуре.
	tagPK = "pk"

	// tagRef — внешний ключ на другую таблицу.
	// Формат: `qqm:"ref=table.column"` или `qqm:"ref=table"`.
	tagRef = "ref="

	// tagPrefix — префикс для колонок из анонимной (embedded) структуры.
	// Пример: `qqm:"prefix=audit_"` — все колонки из embedded struct получат префикс "audit_".
	tagPrefix = "prefix="

	// tagReadonly — поле только для чтения, не участвует в UPDATE.
	// Пример: `qqm:"readonly"`.
	tagReadonly = "readonly"

	// tagAuto — автогенерируемое поле, не участвует в INSERT.
	// Пример: `qqm:"auto"`.
	tagAuto = "auto"

	// tagOmit — поле пропускается при генерации SQL.
	// Пример: `qqm:"omit"`.
	tagOmit = "omit"

	// tagSeparator — разделитель ключей в теге
	tagSeparator = ";"

	// tagName — имя тега для метаданных
	tagName = "qqm"
)

// TagOptions — разобранные опции тега qqm.
type TagOptions struct {
	Col      string
	IsPK     bool
	RefTable string
	RefCol   string
	Prefix   string
	Readonly bool
	Auto     bool
	Omit     bool
}

// Updated at 2026-06-28
// ParseTag разбирает строку тега qqm в TagOptions.
// Формат: "col=name;pk;ref=users.id;prefix=audit_;readonly;auto;omit"
// Разделитель: точка с запятой (;).
// Тег "pk" — флаг, порядок определяется порядком объявления в структуре.
func ParseTag(raw string) TagOptions {
	var opts TagOptions
	if raw == "" {
		return opts
	}

	parts := strings.Split(raw, tagSeparator)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		switch {
		case strings.HasPrefix(p, tagCol):
			opts.Col = strings.TrimPrefix(p, tagCol)
		case p == tagPK:
			opts.IsPK = true
		case strings.HasPrefix(p, tagRef):
			ref := strings.TrimPrefix(p, tagRef)
			if dot := strings.IndexByte(ref, '.'); dot >= 0 {
				opts.RefTable = ref[:dot]
				opts.RefCol = ref[dot+1:]
			} else {
				opts.RefTable = ref
			}
		case strings.HasPrefix(p, tagPrefix):
			opts.Prefix = strings.TrimPrefix(p, tagPrefix)
		case p == tagReadonly:
			opts.Readonly = true
		case p == tagAuto:
			opts.Auto = true
		case p == tagOmit:
			opts.Omit = true
		}
	}

	return opts
}
