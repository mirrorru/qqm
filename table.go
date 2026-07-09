package qqm

import (
	"context"
	"reflect"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/meta"
	"github.com/mirrorru/qqm/txproc"
)

// SQLNamer описывает интерфейс для получения имени таблицы из структуры.
// Реализуйте этот интерфейс, если имя таблицы отличается от snake_case имени типа.
// EN: SQLNamer describes the interface for getting a table name from a struct.
// Implement this interface when the table name differs from the snake_case type name.
type SQLNamer interface {
	SQLName() string
}

// CRUD describes CRUD operations for a table.
// EN: CRUD описывает операции CRUD для таблицы.
type CRUD[ROW any] interface {
	Internals() *tableInternals

	Insert(ctx context.Context, ex txproc.TxProcessor, src *ROW) (*ROW, error)
	Update(ctx context.Context, ex txproc.TxProcessor, src *ROW) (*ROW, error)
	GetByPK(ctx context.Context, ex txproc.TxProcessor, keys ...any) (*ROW, error)
	Delete(ctx context.Context, ex txproc.TxProcessor, keys ...any) error
	List(ctx context.Context, ex txproc.TxProcessor, filters ...Filter) ([]*ROW, error)
}

// tableInternals содержит внутренние данные таблицы для реализации CRUD.
// EN: tableInternals holds table internals for CRUD implementation.
type tableInternals struct {
	dialect      dialect.DialectProvider
	meta         *meta.RowMeta
	queries      *queryBuilder
	whereBuilder *whereBuilder
	scanHelper   *scanDestHelper
}

// scanDestHelper хранит индексы полей и подготовленные dest для Scan.
// Позволяет переиспользовать слайс dest для разных строк.
// EN: scanDestHelper holds field indexes and prepared dest for Scan.
// Allows reusing dest slice for different rows.
type scanDestHelper struct {
	fieldIndexes [][]int
	dest         []any
}

// newScanDestHelper создаёт scanDestHelper для RowMeta.
// Заполняет индексы всех не-omit полей.
// EN: newScanDestHelper creates a scanDestHelper for RowMeta.
// Fills indexes of all non-omit fields.
func newScanDestHelper(rm *meta.RowMeta) *scanDestHelper {
	var indexes [][]int
	for _, fm := range rm.Fields {
		if fm.IsOmit {
			continue
		}
		idx := make([]int, len(fm.Index))
		copy(idx, fm.Index)
		indexes = append(indexes, idx)
	}
	return &scanDestHelper{
		fieldIndexes: indexes,
		dest:         make([]any, len(indexes)),
	}
}

// resetForRow обновляет dest указатели на поля нового row-значения.
// Возвращает готовый dest слайс для Scan.
// EN: resetForRow updates dest pointers to fields of a new row value.
// Returns the ready dest slice for Scan.
func (h *scanDestHelper) resetForRow(rowVal reflect.Value) []any {
	for i, idx := range h.fieldIndexes {
		fv := rowVal.FieldByIndex(idx)
		h.dest[i] = fv.Addr().Interface()
	}
	return h.dest
}

// Table represents a database table with a typed ROW struct.
// EN: Table представляет таблицу БД с типизированной структурой ROW.
type Table[ROW any] struct {
	internal tableInternals
	rowType  reflect.Type
}

// NewTableVal создаёт Table[ROW] для указанного диалекта.
// Возвращает значение (не указатель), рекомендуется использовать через указатель.
// EN: NewTableVal creates a Table[ROW] for the specified dialect.
// Returns a value (not pointer), recommended to use via pointer.
func NewTableVal[ROW any](d dialect.DialectProvider) Table[ROW] {
	var zero ROW
	rt := reflect.TypeOf(zero)

	if rt.Kind() == reflect.Pointer {
		panic("qqm: ROW must not be a pointer type, use struct value")
	}

	tableName := resolveTableName(rt, zero)
	rm := meta.GetOrBuildRowMeta(rt, tableName)

	elemType := rt
	for elemType.Kind() == reflect.Pointer {
		elemType = elemType.Elem()
	}

	return Table[ROW]{
		internal: tableInternals{
			dialect:      d,
			meta:         rm,
			queries:      newQueryBuilder(),
			whereBuilder: newWhereBuilder(d, rm.Fields),
			scanHelper:   newScanDestHelper(rm),
		},
		rowType: elemType,
	}
}

// NewTable создаёт указатель на Table[ROW] для указанного диалекта.
// EN: NewTable creates a pointer to Table[ROW] for the specified dialect.
func NewTable[ROW any](d dialect.DialectProvider) *Table[ROW] {
	return new(NewTableVal[ROW](d))
}

// resolveTableName определяет имя таблицы для типа ROW.
// Приоритет: SQLNamer > snake_case(TypeName).
// EN: resolveTableName determines the table name for the ROW type.
// Priority: SQLNamer > snake_case(TypeName).
func resolveTableName[ROW any](rt reflect.Type, zero ROW) string {
	base := rt
	for base.Kind() == reflect.Pointer {
		base = base.Elem()
	}

	if namer, ok := any(zero).(SQLNamer); ok {
		return namer.SQLName()
	}

	if rt.Kind() != reflect.Pointer {
		ptrVal := reflect.New(rt)
		if namer, ok := ptrVal.Interface().(SQLNamer); ok {
			return namer.SQLName()
		}
	}

	return meta.ToSnakeCase(base.Name())
}

// Internals возвращает внутренние данные таблицы (для доступа к SQL и метаданным).
// EN: Internals returns the table internals (for access to SQL and metadata).
func (t *Table[ROW]) Internals() *tableInternals {
	return &t.internal
}

// Meta возвращает метаданные строки ROW.
// EN: Meta returns the ROW struct metadata.
func (i *tableInternals) Meta() *meta.RowMeta {
	return i.meta
}

// Dialect возвращает диалект БД.
// EN: Dialect returns the DB dialect.
func (i *tableInternals) Dialect() dialect.DialectProvider {
	return i.dialect
}

// InsertSQL возвращает кэшированный SQL INSERT для таблицы.
// EN: InsertSQL returns the cached INSERT SQL for the table.
func (i *tableInternals) InsertSQL() string {
	return i.queries.InsertSQL(i.dialect, i.meta)
}

// UpdateSQL возвращает кэшированный SQL UPDATE для таблицы.
// EN: UpdateSQL returns the cached UPDATE SQL for the table.
func (i *tableInternals) UpdateSQL() string {
	return i.queries.UpdateSQL(i.dialect, i.meta)
}

// SelectSQL возвращает кэшированный SQL SELECT для таблицы.
// EN: SelectSQL returns the cached SELECT SQL for the table.
func (i *tableInternals) SelectSQL() string {
	return i.queries.SelectSQL(i.dialect, i.meta)
}

// DeleteSQL возвращает кэшированный SQL DELETE для таблицы.
// EN: DeleteSQL returns the cached DELETE SQL for the table.
func (i *tableInternals) DeleteSQL() string {
	return i.queries.DeleteSQL(i.dialect, i.meta)
}

// ListSQL возвращает кэшированный SQL SELECT ALL для таблицы.
// EN: ListSQL returns the cached SELECT ALL SQL for the table.
func (i *tableInternals) ListSQL() string {
	return i.queries.ListSQL(i.dialect, i.meta)
}

// CreateTableSQL возвращает кэшированный SQL CREATE TABLE для таблицы.
// EN: CreateTableSQL returns the cached CREATE TABLE SQL for the table.
func (i *tableInternals) CreateTableSQL() string {
	return i.queries.CreateTableSQL(i.dialect, i.meta)
}

// Insert вставляет новую строку и возвращает её (с заполненными автогенерируемыми полями).
// EN: Insert inserts a new row and returns it (with auto-generated fields populated).
func (t *Table[ROW]) Insert(ctx context.Context, ex txproc.TxProcessor, src *ROW) (*ROW, error) {
	args := t.internal.meta.InsertValues(src)

	if t.internal.dialect.SupportsReturning() {
		row := ex.QueryRowContext(ctx, t.internal.InsertSQL(), args...)
		buf := new(ROW)
		dest := t.internal.scanHelper.resetForRow(t.rowValue(buf))
		if err := row.Scan(dest...); err != nil {
			return nil, err
		}
		result := new(ROW)
		*result = *buf
		return result, nil
	}

	_, err := ex.ExecContext(ctx, t.internal.InsertSQL(), args...)
	if err != nil {
		return nil, err
	}
	result := new(ROW)
	*result = *src
	return result, nil
}

// Update обновляет существующую строку и возвращает её.
// EN: Update updates an existing row and returns it.
func (t *Table[ROW]) Update(ctx context.Context, ex txproc.TxProcessor, src *ROW) (*ROW, error) {
	updateVals := t.internal.meta.UpdateValues(src)
	pkVals := t.internal.meta.PKFieldValues(src)
	args := append(updateVals, pkVals...)

	if t.internal.dialect.SupportsReturning() {
		row := ex.QueryRowContext(ctx, t.internal.UpdateSQL(), args...)
		buf := new(ROW)
		dest := t.internal.scanHelper.resetForRow(t.rowValue(buf))
		if err := row.Scan(dest...); err != nil {
			return nil, err
		}
		result := new(ROW)
		*result = *buf
		return result, nil
	}

	_, err := ex.ExecContext(ctx, t.internal.UpdateSQL(), args...)
	if err != nil {
		return nil, err
	}
	result := new(ROW)
	*result = *src
	return result, nil
}

// GetByPK находит строку по первичному ключу.
// EN: GetByPK finds a row by primary key.
func (t *Table[ROW]) GetByPK(ctx context.Context, ex txproc.TxProcessor, keys ...any) (*ROW, error) {
	row := ex.QueryRowContext(ctx, t.internal.SelectSQL(), keys...)

	buf := new(ROW)
	dest := t.internal.scanHelper.resetForRow(t.rowValue(buf))
	err := row.Scan(dest...)
	return buf, err
}

// Delete удаляет строку по первичному ключу.
// EN: Delete deletes a row by primary key.
func (t *Table[ROW]) Delete(ctx context.Context, ex txproc.TxProcessor, keys ...any) error {
	_, err := ex.ExecContext(ctx, t.internal.DeleteSQL(), keys...)
	return err
}

// List возвращает все строки таблицы (с необязательными фильтрами).
// EN: List returns all rows from the table (with optional filters).
func (t *Table[ROW]) List(ctx context.Context, ex txproc.TxProcessor, filters ...Filter) ([]*ROW, error) {
	if len(filters) == 0 {
		rows, err := ex.QueryContext(ctx, t.internal.ListSQL())
		if err != nil {
			return nil, err
		}
		defer func() { _ = rows.Close() }()

		var result []*ROW
		buf := new(ROW)
		dest := t.internal.scanHelper.resetForRow(t.rowValue(buf))
		for rows.Next() {
			if err := rows.Scan(dest...); err != nil {
				return nil, err
			}
			row := new(ROW)
			*row = *buf
			result = append(result, row)
		}
		return result, nil
	}

	sql, args, err := t.buildFilterWhereClause(filters)
	if err != nil {
		return nil, err
	}

	query := t.internal.ListSQL() + sql
	rows, err := ex.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []*ROW
	for rows.Next() {
		buf := new(ROW)
		dest := t.internal.scanHelper.resetForRow(t.rowValue(buf))
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		row := new(ROW)
		*row = *buf
		result = append(result, row)
	}
	return result, nil
}

// rowValue возвращает reflect.Value структуры по указателю.
// EN: rowValue returns the reflect.Value of struct from a pointer.
func (t *Table[ROW]) rowValue(row *ROW) reflect.Value {
	return reflect.ValueOf(row).Elem()
}

// buildFilterWhereClause формирует WHERE-условие и аргументы из фильтров.
// EN: buildFilterWhereClause builds the WHERE clause and arguments from filters.
func (t *Table[ROW]) buildFilterWhereClause(filters []Filter) (string, []any, error) {
	return t.internal.whereBuilder.buildWhereClause(filters)
}
