// Updated at 2026-06-29
package meta

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

	// tagJoin — тип JOIN (LEFT, INNER, RIGHT, FULL).
	// Пример: `qqm:"join=LEFT"`.
	tagJoin = "join="

	// tagPrimary — явное указание первичной таблицы в Query.
	// Пример: `qqm:"primary"`.
	tagPrimary = "primary"

	// tagOn — явное условие JOIN.
	// Пример: `qqm:"on=users.id=orders.user_id"`.
	tagOn = "on="

	// tagTable — переопределение имени таблицы для поля в Query.
	// Пример: `qqm:"table=app_users"`.
	tagTable = "table="

	// tagName — имя тега для метаданных
	tagName = "qqm"
)

// TagOptions — разобранные опции тега qqm.
type TagOptions struct {
	Col       string
	IsPK      bool
	RefTable  string
	RefCol    string
	Prefix    string
	Readonly  bool
	Auto      bool
	Omit      bool
	JoinType  string
	IsPrimary bool
	On        string
	TableName string
}

// Updated at 2026-06-29
// ParseTag разбирает строку тега qqm в TagOptions.
// Формат: "col=name;pk;ref=users.id;prefix=audit_;readonly;auto;omit;join=LEFT;primary;on=cond"
// Разделитель: точка с запятой (;).
// Тег "pk" — флаг, порядок определяется порядком объявления в структуре.
func ParseTag(raw string) TagOptions {
	var opts TagOptions
	if raw == "" {
		return opts
	}

	i := 0
	n := len(raw)
	for i < n {
		// пропускаем пробелы в начале сегмента
		for i < n && raw[i] == ' ' {
			i++
		}
		if i >= n {
			break
		}

		start := i
		for i < n && raw[i] != ';' {
			i++
		}

		seg := raw[start:i]
		// trim trailing spaces
		end := len(seg)
		for end > 0 && seg[end-1] == ' ' {
			end--
		}
		seg = seg[:end]

		if seg == "" {
			if i < n {
				i++ // skip ;
			}
			continue
		}

		switch {
		case len(seg) > 4 && seg[:4] == tagCol:
			opts.Col = seg[4:]
		case seg == tagPK:
			opts.IsPK = true
		case len(seg) > 4 && seg[:4] == tagRef:
			ref := seg[4:]
			if dot := indexByte(ref, '.'); dot >= 0 {
				opts.RefTable = ref[:dot]
				opts.RefCol = ref[dot+1:]
			} else {
				opts.RefTable = ref
			}
		case len(seg) > 7 && seg[:7] == tagPrefix:
			opts.Prefix = seg[7:]
		case seg == tagReadonly:
			opts.Readonly = true
		case seg == tagAuto:
			opts.Auto = true
		case seg == tagOmit:
			opts.Omit = true
		case len(seg) > 5 && seg[:5] == tagJoin:
			opts.JoinType = seg[5:]
		case seg == tagPrimary:
			opts.IsPrimary = true
		case len(seg) > 3 && seg[:3] == tagOn:
			opts.On = seg[3:]
		case len(seg) > 6 && seg[:6] == tagTable:
			opts.TableName = seg[6:]
		}

		if i < n {
			i++ // skip ;
		}
	}

	return opts
}

// indexByte — как strings.IndexByte, но без импорта.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
