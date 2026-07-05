package meta

const (
	// tagCol — имя колонки в БД. Пример: `qqm:"col=user_name"`.
	// EN: tagCol — column name in DB. Example: `qqm:"col=user_name"`.
	tagCol = "col="

	// tagPK — поле является частью первичного ключа.
	// Пример: `qqm:"pk"`. Порядок — по объявлению в структуре.
	// EN: tagPK — field is part of the primary key.
	// Example: `qqm:"pk"`. Order is by declaration in the struct.
	tagPK = "pk"

	// tagRef — внешний ключ на другую таблицу.
	// Формат: `qqm:"ref=table.column"` или `qqm:"ref=table"`.
	// EN: tagRef — foreign key reference to another table.
	// Format: `qqm:"ref=table.column"` or `qqm:"ref=table"`.
	tagRef = "ref="

	// tagPrefix — префикс для колонок из анонимной (embedded) структуры.
	// Пример: `qqm:"prefix=audit_"` добавляет префикс "audit_" ко всем колонкам embedded struct.
	// EN: tagPrefix — prefix for columns from an anonymous (embedded) struct.
	// Example: `qqm:"prefix=audit_"` adds the "audit_" prefix to all columns from the embedded struct.
	tagPrefix = "prefix="

	// tagUpdate — разрешает UPDATE для поля с тегом auto. Пример: `qqm:"auto;update"`.
	// EN: tagUpdate — allows UPDATE for a field with the auto tag. Example: `qqm:"auto;update"`.
	tagUpdate = "update"

	// tagAuto — автогенерируемое поле, исключается из INSERT. Пример: `qqm:"auto"`.
	// EN: tagAuto — auto-generated field, excluded from INSERT. Example: `qqm:"auto"`.
	tagAuto = "auto"

	// tagOmit — поле пропускается при генерации SQL. Пример: `qqm:"omit"`.
	// EN: tagOmit — field is skipped during SQL generation. Example: `qqm:"omit"`.
	tagOmit = "omit"

	// tagJoin — тип JOIN (LEFT, INNER, RIGHT, FULL). Пример: `qqm:"join=LEFT"`.
	// EN: tagJoin — JOIN type (LEFT, INNER, RIGHT, FULL). Example: `qqm:"join=LEFT"`.
	tagJoin = "join="

	// tagPrimary — явное указание первичной таблицы в Query. Пример: `qqm:"primary"`.
	// EN: tagPrimary — explicit primary table marker in Query. Example: `qqm:"primary"`.
	tagPrimary = "primary"

	// tagTable — переопределение имени таблицы для поля в Query. Пример: `qqm:"table=app_users"`.
	// EN: tagTable — override table name for a field in Query. Example: `qqm:"table=app_users"`.
	tagTable = "table="

	// tagSort — порядок сортировки в List(). Формат: `qqm:"sort=<position>[,direction]"`.
	// position — порядок поля в сортировке (1-based).
	// direction — ASC (по умолчанию) или DESC.
	// EN: tagSort — sort order in List(). Format: `qqm:"sort=<position>[,direction]"`.
	// position — field order in sorting (1-based).
	// direction — ASC (default) or DESC.
	tagSort = "sort="

	// tagCreate — строка для колонки в CREATE TABLE (DEFAULT, UNIQUE, CHECK и т.д.).
	// Формат: `qqm:"create=DEFAULT 0"` или `qqm:"create=UNIQUE"`.
	// EN: tagCreate — column definition in CREATE TABLE (DEFAULT, UNIQUE, CHECK, etc.).
	// Format: `qqm:"create=DEFAULT 0"` or `qqm:"create=UNIQUE"`.
	tagCreate = "create="

	// tagInsert — поле участвует в INSERT, но исключается из UPDATE.
	// Пример: `qqm:"insert"`. Аналог: created_at — задаётся при создании, не меняется при обновлении.
	// EN: tagInsert — field participates in INSERT but is excluded from UPDATE.
	// Example: `qqm:"insert"`. Analog: created_at — set on create, not changed on update.
	tagInsert = "insert"
)

// TagName содержит имя тега для метаданных.
// EN: TagName holds the tag name for metadata.
var TagName = "qqm"

// TagOptions содержит разобранные опции тега qqm.
// EN: TagOptions holds parsed options of the qqm tag.
type TagOptions struct {
	Col       string // Имя колонки (из col=). / EN: Column name (from col=).
	IsPK      bool   // Является ли первичным ключом. / EN: Is primary key.
	RefTable  string // Таблица для FK (из ref=). / EN: Table for FK (from ref=).
	RefCol    string // Колонка для FK. / EN: Column for FK.
	Prefix    string // Префикс колонок (из prefix=). / EN: Column prefix (from prefix=).
	Update    bool   // Разрешает UPDATE для auto-полей. / EN: Allows UPDATE for auto fields.
	Auto      bool   // Автогенерируемое поле. / EN: Auto-generated field.
	Omit      bool   // Пропустить при генерации SQL. / EN: Skip during SQL generation.
	JoinType  string // Тип JOIN (из join=). / EN: JOIN type (from join=).
	IsPrimary bool   // Явно помечена как первичная. / EN: Explicitly marked as primary.
	TableName string // Переопределённое имя таблицы (из table=). / EN: Overridden table name (from table=).
	Sort      int    // Позиция в сортировке (0 если не задана). / EN: Position in ordering (0 if not set).
	SortDir   string // Направление: "ASC" (по умолчанию) или "DESC". / EN: Sort direction: "ASC" (default) or "DESC".
	Create    string // Строка для CREATE TABLE (из create=...). / EN: Column definition for CREATE TABLE (from create=...).
	Insert    bool   // Участвует в INSERT, исключается из UPDATE. / EN: Participates in INSERT, excluded from UPDATE.
}

// ParseTag разбирает строку тега qqm в TagOptions.
// Формат: "col=name;pk;ref=users.id;prefix=audit_;update;auto;omit;join=LEFT;primary;insert"
// Разделитель — точка с запятой (;). Тег "pk" — флаг, порядок определяется объявлением в структуре.
// EN: ParseTag parses the qqm tag string into TagOptions.
// Format: "col=name;pk;ref=users.id;prefix=audit_;update;auto;omit;join=LEFT;primary;insert"
// Separator — semicolon (;). "pk" tag is a flag; order is by struct declaration.
func ParseTag(raw string) TagOptions {
	var opts TagOptions
	if raw == "" {
		return opts
	}

	i := 0
	n := len(raw)
	for i < n {
		// Пропускаем пробелы в начале сегмента.
		// EN: Skip leading spaces.
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
		// Trim trailing spaces.
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
			if dot := lastIndexByte(ref, '.'); dot >= 0 {
				opts.RefTable = ref[:dot]
				opts.RefCol = ref[dot+1:]
			} else {
				opts.RefTable = ref
			}
		case len(seg) > 7 && seg[:7] == tagPrefix:
			opts.Prefix = seg[7:]
		case seg == tagUpdate:
			opts.Update = true
		case seg == tagAuto:
			opts.Auto = true
		case seg == tagOmit:
			opts.Omit = true
		case len(seg) > 5 && seg[:5] == tagJoin:
			opts.JoinType = seg[5:]
		case seg == tagPrimary:
			opts.IsPrimary = true
		case len(seg) > 6 && seg[:6] == tagTable:
			opts.TableName = seg[6:]
		case len(seg) > 5 && seg[:5] == tagSort:
			opts.Sort, opts.SortDir = parseSortSegment(seg[5:])
		case len(seg) > 7 && seg[:7] == tagCreate:
			opts.Create = seg[7:]
		case seg == tagInsert:
			opts.Insert = true
		}

		if i < n {
			i++ // skip ;
		}
	}

	return opts
}

// parseSortSegment разбирает значение после "sort=": "<position>[,direction]".
// EN: parseSortSegment parses the value after "sort=": "<position>[,direction]".
func parseSortSegment(raw string) (pos int, dir string) {
	if raw == "" {
		return 0, ""
	}

	// Ищем запятую.
	// EN: Look for comma.
	comma := indexByte(raw, ',')
	if comma < 0 {
		pos = atoi(raw)
		return pos, "ASC"
	}

	pos = atoi(raw[:comma])
	dirRaw := raw[comma+1:]
	switch {
	case len(dirRaw) >= 3 && (dirRaw[0] == 'A' || dirRaw[0] == 'a') &&
		(dirRaw[1] == 'S' || dirRaw[1] == 's') &&
		(dirRaw[2] == 'C' || dirRaw[2] == 'c'):
		dir = "ASC"
	case len(dirRaw) >= 4 && (dirRaw[0] == 'D' || dirRaw[0] == 'd') &&
		(dirRaw[1] == 'E' || dirRaw[1] == 'e') &&
		(dirRaw[2] == 'S' || dirRaw[2] == 's') &&
		(dirRaw[3] == 'C' || dirRaw[3] == 'c'):
		dir = "DESC"
	default:
		return 0, ""
	}
	return pos, dir
}

// atoi выполняет быстрый парсинг целого числа без импорта strconv.
// EN: atoi performs fast integer parsing without strconv import.
func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0
		}
		n = n*10 + int(s[i]-'0')
	}
	return n
}

// indexByte — аналог strings.IndexByte без импорта strings.
// EN: indexByte — strings.IndexByte equivalent without importing strings.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// lastIndexByte — аналог strings.LastIndexByte без импорта strings.
// EN: lastIndexByte — strings.LastIndexByte equivalent without importing strings.
func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
