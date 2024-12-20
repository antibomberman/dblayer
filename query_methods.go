package dblayer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// DateFunctions содержит SQL функции для разных СУБД
type DateFunctions struct {
	DateDiff    string
	DateTrunc   string
	DateFormat  string
	TimeZone    string
	Extract     string
	DateAdd     string
	CurrentDate string
}
type PaginationResult struct {
	Data        interface{} `json:"data"`
	Total       int64       `json:"total"`
	PerPage     int         `json:"per_page"`
	CurrentPage int         `json:"current_page"`
	LastPage    int         `json:"last_page"`
}

// Select указывает колонки для выборки
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.columns = columns
	return qb
}

// Where добавляет условие AND
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   condition,
		args:     args,
	})
	return qb
}

// WhereId добавляет условие WHERE id = ?
func (qb *QueryBuilder) WhereId(id interface{}) *QueryBuilder {
	qb.Where("id = ?", id)
	return qb
}

// OrWhere добавляет условие OR
func (qb *QueryBuilder) OrWhere(condition string, args ...interface{}) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "OR",
		clause:   condition,
		args:     args,
	})
	return qb
}

// WhereIn добавляет условие IN
func (qb *QueryBuilder) WhereIn(column string, values ...interface{}) *QueryBuilder {
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "?"
	}
	condition := fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ","))
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   condition,
		args:     values,
	})
	return qb
}

// WhereGroup добавляет группу условий
func (qb *QueryBuilder) WhereGroup(fn func(*QueryBuilder)) *QueryBuilder {
	group := &QueryBuilder{}
	fn(group)
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		nested:   group.conditions,
	})
	return qb
}

// OrWhereGroup добавляет группу условий через OR
func (qb *QueryBuilder) OrWhereGroup(fn func(*QueryBuilder)) *QueryBuilder {
	group := &QueryBuilder{}
	fn(group)
	qb.conditions = append(qb.conditions, Condition{
		operator: "OR",
		nested:   group.conditions,
	})
	return qb
}

// Join добавляет INNER JOIN
func (qb *QueryBuilder) Join(table string, condition string) *QueryBuilder {
	qb.joins = append(qb.joins, Join{
		Type:      InnerJoin,
		Table:     table,
		Condition: condition,
	})
	return qb
}

// LeftJoin добавляет LEFT JOIN
func (qb *QueryBuilder) LeftJoin(table string, condition string) *QueryBuilder {
	qb.joins = append(qb.joins, Join{
		Type:      LeftJoin,
		Table:     table,
		Condition: condition,
	})
	return qb
}

// RightJoin добавляет RIGHT JOIN
func (qb *QueryBuilder) RightJoin(table string, condition string) *QueryBuilder {
	qb.joins = append(qb.joins, Join{
		Type:      RightJoin,
		Table:     table,
		Condition: condition,
	})
	return qb
}

// CrossJoin добавляет CROSS JOIN
func (qb *QueryBuilder) CrossJoin(table string) *QueryBuilder {
	qb.joins = append(qb.joins, Join{
		Type:  CrossJoin,
		Table: table,
	})
	return qb
}

// OrderBy добавляет сортировку
func (qb *QueryBuilder) OrderBy(column string, direction string) *QueryBuilder {
	qb.orderBy = append(qb.orderBy, fmt.Sprintf("%s %s", column, direction))
	return qb
}

// GroupBy добавляет группировку
func (qb *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
	qb.groupBy = columns
	return qb
}

// Having добавляет условие для группировки
func (qb *QueryBuilder) Having(condition string) *QueryBuilder {
	qb.having = condition
	return qb
}

// Limit устанавливает ограничение на количество записей
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset устанавливает смещение
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// As устанавливает алиас для таблицы
func (qb *QueryBuilder) As(alias string) *QueryBuilder {
	qb.alias = alias
	return qb
}

// Find ищет запись по id
func (qb *QueryBuilder) Find(id interface{}, dest interface{}) (bool, error) {
	qb.Where("id = ?", id)
	return qb.First(dest)
}

// FindContext ищет запись по id с контекстом
func (qb *QueryBuilder) FindContext(ctx context.Context, id interface{}, dest interface{}) (bool, error) {
	qb.Where("id = ?", id)
	return qb.FirstContext(ctx, dest)
}

// Increment увеличивает значение поля
func (qb *QueryBuilder) Increment(column string, value interface{}) error {
	var args []interface{}
	args = append(args, value)

	query := fmt.Sprintf("UPDATE %s SET %s = %s + ?", qb.table, column, column)

	if len(qb.conditions) > 0 {
		whereSQL := buildConditions(qb.conditions)
		query += " WHERE " + whereSQL
		// Добавляем аргументы из условий WHERE
		for _, cond := range qb.conditions {
			args = append(args, cond.args...)
		}
	}

	return qb.execExec(query, args...)
}

// Decrement уменьшает значение поля
func (qb *QueryBuilder) Decrement(column string, value interface{}) error {
	var args []interface{}
	args = append(args, value)

	query := fmt.Sprintf("UPDATE %s SET %s = %s - ?", qb.table, column, column)

	if len(qb.conditions) > 0 {
		whereSQL := buildConditions(qb.conditions)
		query += " WHERE " + whereSQL
		// Добавляем аргументы из условий WHERE
		for _, cond := range qb.conditions {
			args = append(args, cond.args...)
		}
	}

	return qb.execExec(query, args...)
}

// Get получает все записи
func (qb *QueryBuilder) Get(dest interface{}) (bool, error) {
	query, args := qb.buildQuery()
	return qb.execSelect(dest, query, args...)
}

// GetContext получает все записи с контекстом
func (qb *QueryBuilder) GetContext(ctx context.Context, dest interface{}) (bool, error) {
	query, args := qb.buildQuery()
	return qb.execSelectContext(ctx, dest, query, args...)
}

// First получает первую запись
func (qb *QueryBuilder) First(dest interface{}) (bool, error) {
	qb.Limit(1)
	query, args := qb.buildQuery()
	return qb.execGet(dest, query, args...)
}

// FirstContext получает первую запись с контекстом
func (qb *QueryBuilder) FirstContext(ctx context.Context, dest interface{}) (bool, error) {
	qb.Limit(1)
	query, args := qb.buildQuery()
	return qb.execGetContext(ctx, dest, query, args...)
}

// Create создает новую запись из структуры и возвращает её id
func (qb *QueryBuilder) Create(data interface{}) (int64, error) {
	fields, placeholders := qb.getStructInfo(data)
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		qb.table,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "))

	if qb.getDriverName() == "postgres" {
		var id int64
		query += " RETURNING id"
		err := qb.getExecutor().QueryRowx(query, data).Scan(&id)
		return id, err
	}

	result, err := qb.getExecutor().NamedExec(query, data)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// CreateContext создает новую запись из структуры и возвращает её id
func (qb *QueryBuilder) CreateContext(ctx context.Context, data interface{}) (int64, error) {
	fields, placeholders := qb.getStructInfo(data)

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		qb.table,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "))

	if qb.getDriverName() == "postgres" {
		var id int64
		query += " RETURNING id"
		err := qb.getExecutor().(sqlx.QueryerContext).QueryRowxContext(ctx, query, data).Scan(&id)
		return id, err
	}

	result, err := qb.getExecutor().NamedExecContext(ctx, query, data)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// CreateMap создает новую запись из map и возвращает её id
func (qb *QueryBuilder) CreateMap(data map[string]interface{}) (int64, error) {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		qb.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	if qb.getDriverName() == "postgres" {
		var id int64
		query = qb.rebindQuery(query + " RETURNING id")
		err := qb.getExecutor().QueryRowx(query, values...).Scan(&id)
		return id, err
	}

	result, err := qb.getExecutor().Exec(qb.rebindQuery(query), values...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// CreateMapContext создает новую запись из map с контекстом и возвращает её id
func (qb *QueryBuilder) CreateMapContext(ctx context.Context, data map[string]interface{}) (int64, error) {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		qb.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	if qb.getDriverName() == "postgres" {
		var id int64
		query = qb.rebindQuery(query + " RETURNING id")
		err := qb.getExecutor().(sqlx.QueryerContext).QueryRowxContext(ctx, query, values...).Scan(&id)
		return id, err
	}

	result, err := qb.getExecutor().ExecContext(ctx, qb.rebindQuery(query), values...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// BatchInsert вставляет множество записей
func (qb *QueryBuilder) BatchInsert(records []map[string]interface{}) error {
	if len(records) == 0 {
		return nil
	}

	// Получаем все колонки из первой записи
	columns := make([]string, 0)
	for column := range records[0] {
		columns = append(columns, column)
	}
	sort.Strings(columns)

	// Создаем placeholders и значения
	var placeholders []string
	var values []interface{}
	for _, record := range records {
		placeholder := make([]string, len(columns))
		for i := range columns {
			placeholder[i] = "?"
			values = append(values, record[columns[i]])
		}
		placeholders = append(placeholders, "("+strings.Join(placeholder, ", ")+")")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		qb.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	return qb.execExec(query, values...)
}

// BatchInsertContext вставляет множество записей с контекстом
func (qb *QueryBuilder) BatchInsertContext(ctx context.Context, records []map[string]interface{}) error {
	if len(records) == 0 {
		return nil
	}

	// Получаем все колонки из первой записи
	columns := make([]string, 0)
	for column := range records[0] {
		columns = append(columns, column)
	}
	sort.Strings(columns)

	// Создаем placeholders и значения
	var placeholders []string
	var values []interface{}
	for _, record := range records {
		placeholder := make([]string, len(columns))
		for i := range columns {
			placeholder[i] = "?"
			values = append(values, record[columns[i]])
		}
		placeholders = append(placeholders, "("+strings.Join(placeholder, ", ")+")")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		qb.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	return qb.execExecContext(ctx, query, values...)
}

// BulkInsert выполняет массовую вставку записей с возвратом ID
func (qb *QueryBuilder) BulkInsert(records []map[string]interface{}) ([]int64, error) {
	if len(records) == 0 {
		return nil, nil
	}

	// Получаем все колонки из первой записи
	columns := make([]string, 0)
	for column := range records[0] {
		columns = append(columns, column)
	}
	sort.Strings(columns)

	// Создаем placeholders и значения
	var placeholders []string
	var values []interface{}
	for _, record := range records {
		placeholder := make([]string, len(columns))
		for i := range columns {
			placeholder[i] = "?"
			values = append(values, record[columns[i]])
		}
		placeholders = append(placeholders, "("+strings.Join(placeholder, ", ")+")")
	}

	var query string
	if qb.getDriverName() == "postgres" {
		query = fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES %s RETURNING id",
			qb.table,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
		)
		var ids []int64
		err := qb.getExecutor().Select(&ids, qb.rebindQuery(query), values...)
		return ids, err
	}

	query = fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		qb.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := qb.getExecutor().Exec(qb.rebindQuery(query), values...)
	if err != nil {
		return nil, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	ids := make([]int64, rowsAffected)
	for i := range ids {
		ids[i] = lastID + int64(i)
	}

	return ids, nil
}

// BulkInsertContext выполняет массовую вставку записей с возвратом ID и поддержкой контекста
func (qb *QueryBuilder) BulkInsertContext(ctx context.Context, records []map[string]interface{}) ([]int64, error) {
	if len(records) == 0 {
		return nil, nil
	}

	// Получаем все колонки из первой записи
	columns := make([]string, 0)
	for column := range records[0] {
		columns = append(columns, column)
	}
	sort.Strings(columns)

	// Создаем placeholders и значения
	var placeholders []string
	var values []interface{}
	for _, record := range records {
		placeholder := make([]string, len(columns))
		for i := range columns {
			placeholder[i] = "?"
			values = append(values, record[columns[i]])
		}
		placeholders = append(placeholders, "("+strings.Join(placeholder, ", ")+")")
	}

	var query string
	if qb.getDriverName() == "postgres" {
		query = fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES %s RETURNING id",
			qb.table,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
		)
		var ids []int64
		err := qb.getExecutor().SelectContext(ctx, &ids, qb.rebindQuery(query), values...)
		return ids, err
	}

	query = fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		qb.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := qb.getExecutor().(sqlx.ExtContext).ExecContext(ctx, qb.rebindQuery(query), values...)
	if err != nil {
		return nil, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	ids := make([]int64, rowsAffected)
	for i := range ids {
		ids[i] = lastID + int64(i)
	}

	return ids, nil
}

// Update обновляет записи используя структуру
func (qb *QueryBuilder) Update(data interface{}, fields ...string) error {
	var sets []string
	if len(fields) > 0 {
		// Обновляем только указанные поля
		for _, field := range fields {
			sets = append(sets, fmt.Sprintf("%s = :%s", field, field))
		}
	} else {
		// Обновляем все поля
		fields, _ := qb.getStructInfo(data)
		for _, field := range fields {
			sets = append(sets, fmt.Sprintf("%s = :%s", field, field))
		}
	}

	whereSQL := buildConditions(qb.conditions)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		qb.table,
		strings.Join(sets, ", "),
		whereSQL)

	_, err := qb.getExecutor().NamedExec(query, data)
	return err
}

// UpdateContext обновляет записи с контекстом
func (qb *QueryBuilder) UpdateContext(ctx context.Context, data interface{}, fields ...string) error {
	var sets []string
	if len(fields) > 0 {
		// Обновляем только указанные поля
		for _, field := range fields {
			sets = append(sets, fmt.Sprintf("%s = :%s", field, field))
		}
	} else {
		// Обновляем все поля
		fields, _ := qb.getStructInfo(data)
		for _, field := range fields {
			sets = append(sets, fmt.Sprintf("%s = :%s", field, field))
		}
	}

	whereSQL := buildConditions(qb.conditions)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		qb.table,
		strings.Join(sets, ", "),
		whereSQL)

	_, err := qb.getExecutor().NamedExecContext(ctx, query, data)
	return err
}

// UpdateMap обновляет записи используя map
func (qb *QueryBuilder) UpdateMap(data map[string]interface{}) error {
	sets := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		sets = append(sets, col+" = ?")
		values = append(values, val)
	}

	whereSQL := buildConditions(qb.conditions)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		qb.table,
		strings.Join(sets, ", "),
		whereSQL)

	return qb.execExec(query, values...)
}

func (qb *QueryBuilder) UpdateMapContext(ctx context.Context, data map[string]interface{}) error {
	sets := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		sets = append(sets, col+" = ?")
		values = append(values, val)
	}

	whereSQL := buildConditions(qb.conditions)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		qb.table,
		strings.Join(sets, ", "),
		whereSQL)

	return qb.execExecContext(ctx, query, values...)
}

// BulkUpdate выполняет массовое обновление записей
func (qb *QueryBuilder) BulkUpdate(records []map[string]interface{}, keyColumn string) error {
	if len(records) == 0 {
		return nil
	}

	// Получаем все колонки из первой записи
	columns := make([]string, 0)
	for column := range records[0] {
		if column != keyColumn {
			columns = append(columns, column)
		}
	}
	sort.Strings(columns)

	// Формируем CASE выражения для каждой колонки
	cases := make([]string, len(columns))
	keyValues := make([]interface{}, 0, len(records))
	valueArgs := make([]interface{}, 0, len(records)*len(columns))

	for i, column := range columns {
		whenClauses := make([]string, 0, len(records))
		for _, record := range records {
			if i == 0 {
				keyValues = append(keyValues, record[keyColumn])
			}
			whenClauses = append(whenClauses, "WHEN ? THEN ?")
			valueArgs = append(valueArgs, record[keyColumn], record[column])
		}
		cases[i] = fmt.Sprintf("%s = CASE %s %s END",
			column,
			keyColumn,
			strings.Join(whenClauses, " "),
		)
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s IN (%s)",
		qb.table,
		strings.Join(cases, ", "),
		keyColumn,
		strings.Repeat("?,", len(records)-1)+"?",
	)

	// Объединяем все аргументы
	args := make([]interface{}, 0, len(valueArgs)+len(keyValues))
	args = append(args, valueArgs...)
	args = append(args, keyValues...)

	return qb.execExec(query, args...)
}

// BulkUpdateContext выполняет массовое обновление записей с контекстом
func (qb *QueryBuilder) BulkUpdateContext(ctx context.Context, records []map[string]interface{}, keyColumn string) error {
	if len(records) == 0 {
		return nil
	}

	// Получаем все колонки из первой записи
	columns := make([]string, 0)
	for column := range records[0] {
		if column != keyColumn {
			columns = append(columns, column)
		}
	}
	sort.Strings(columns)

	// Формируем CASE выражения для каждой колонки
	cases := make([]string, len(columns))
	keyValues := make([]interface{}, 0, len(records))
	valueArgs := make([]interface{}, 0, len(records)*len(columns))

	for i, column := range columns {
		whenClauses := make([]string, 0, len(records))
		for _, record := range records {
			if i == 0 {
				keyValues = append(keyValues, record[keyColumn])
			}
			whenClauses = append(whenClauses, "WHEN ? THEN ?")
			valueArgs = append(valueArgs, record[keyColumn], record[column])
		}
		cases[i] = fmt.Sprintf("%s = CASE %s %s END",
			column,
			keyColumn,
			strings.Join(whenClauses, " "),
		)
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s IN (%s)",
		qb.table,
		strings.Join(cases, ", "),
		keyColumn,
		strings.Repeat("?,", len(records)-1)+"?",
	)

	// Объединяем все аргументы
	args := make([]interface{}, 0, len(valueArgs)+len(keyValues))
	args = append(args, valueArgs...)
	args = append(args, keyValues...)

	return qb.execExecContext(ctx, query, args...)
}

// Delete удаляет записи
func (qb *QueryBuilder) Delete() error {
	if len(qb.conditions) == 0 {
		return errors.New("delete without conditions is not allowed")
	}

	whereSQL := buildConditions(qb.conditions)
	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		qb.table,
		whereSQL)

	return qb.execExec(query)
}

// Count возвращает количество записей
func (qb *QueryBuilder) Count() (int64, error) {
	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", qb.table)
	if len(qb.conditions) > 0 {
		whereSQL := buildConditions(qb.conditions)
		query += " WHERE " + whereSQL
		query = qb.rebindQuery(query)
		_, err := qb.execGet(&count, query)
		return count, err
	}
	_, err := qb.execGet(&count, query)
	return count, err
}

// Exists проверяет существование записей
func (qb *QueryBuilder) Exists() (bool, error) {
	count, err := qb.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Paginate выполняет пагинацию результатов
func (qb *QueryBuilder) Paginate(page int, perPage int, dest interface{}) (*PaginationResult, error) {
	total, err := qb.Count()
	if err != nil {
		return nil, err
	}

	lastPage := int(math.Ceil(float64(total) / float64(perPage)))

	qb.Limit(perPage)
	qb.Offset((page - 1) * perPage)

	_, err = qb.Get(dest)
	if err != nil {
		return nil, err
	}

	return &PaginationResult{
		Data:        dest,
		Total:       total,
		PerPage:     perPage,
		CurrentPage: page,
		LastPage:    lastPage,
	}, nil
}

// SubQuery создает подзапрос
func (qb *QueryBuilder) SubQuery(alias string) *QueryBuilder {
	sql, args := qb.buildQuery()
	return &QueryBuilder{
		columns: []string{fmt.Sprintf("(%s) AS %s", sql, alias)},
		db:      qb.db,
		conditions: []Condition{{
			args: args,
		}},
	}
}

// WhereSubQuery добавляет условие с подзапросом
func (qb *QueryBuilder) WhereSubQuery(column string, operator string, subQuery *QueryBuilder) *QueryBuilder {
	sql, args := subQuery.buildQuery()
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("%s %s (%s)", column, operator, sql),
		args:     args,
	})
	return qb
}

// Union объединяет запросы через UNION
func (qb *QueryBuilder) Union(other *QueryBuilder) *QueryBuilder {
	sql1, args1 := qb.buildQuery()
	sql2, args2 := other.buildQuery()

	return &QueryBuilder{
		db:      qb.db,
		columns: []string{fmt.Sprintf("(%s) UNION (%s)", sql1, sql2)},
		conditions: []Condition{{
			args: append(args1, args2...),
		}},
	}
}

// UnionAll объединяет запросы через UNION ALL
func (qb *QueryBuilder) UnionAll(other *QueryBuilder) *QueryBuilder {
	sql1, args1 := qb.buildQuery()
	sql2, args2 := other.buildQuery()

	return &QueryBuilder{
		db:      qb.db,
		columns: []string{fmt.Sprintf("(%s) UNION ALL (%s)", sql1, sql2)},
		conditions: []Condition{{
			args: append(args1, args2...),
		}},
	}
}

// WhereNull добавляет проверку на NULL
func (qb *QueryBuilder) WhereNull(column string) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("%s IS NULL", column),
	})
	return qb
}

// WhereNotNull добавляет проверку на NOT NULL
func (qb *QueryBuilder) WhereNotNull(column string) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("%s IS NOT NULL", column),
	})
	return qb
}

// WhereBetween добавляет условие BETWEEN
func (qb *QueryBuilder) WhereBetween(column string, start, end interface{}) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("%s BETWEEN ? AND ?", column),
		args:     []interface{}{start, end},
	})
	return qb
}

// WhereNotBetween добавляет условие NOT BETWEEN
func (qb *QueryBuilder) WhereNotBetween(column string, start, end interface{}) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("%s NOT BETWEEN ? AND ?", column),
		args:     []interface{}{start, end},
	})
	return qb
}

// HavingRaw добавляет сырое условие HAVING
func (qb *QueryBuilder) HavingRaw(sql string, args ...interface{}) *QueryBuilder {
	if qb.having != "" {
		qb.having += " AND "
	}
	qb.having += sql
	qb.conditions = append(qb.conditions, Condition{args: args})
	return qb
}

// WithTransaction выполняет запрос в существующей транзакции
func (qb *QueryBuilder) WithTransaction(tx *Transaction) *QueryBuilder {
	qb.db = tx.tx
	return qb
}

// LockForUpdate блокирует записи для обновления
func (qb *QueryBuilder) LockForUpdate() *QueryBuilder {
	return qb.Lock("FOR UPDATE")
}

// LockForShare блокирует записи для чтения
func (qb *QueryBuilder) LockForShare() *QueryBuilder {
	return qb.Lock("FOR SHARE")
}

// SkipLocked пропускает заблокированные записи
func (qb *QueryBuilder) SkipLocked() *QueryBuilder {
	return qb.Lock("SKIP LOCKED")
}

// NoWait не ждет разблокировки записей
func (qb *QueryBuilder) NoWait() *QueryBuilder {
	return qb.Lock("NOWAIT")
}

// Window добавляет оконную функцию
func (qb *QueryBuilder) Window(column string, partition string, orderBy string) *QueryBuilder {
	windowFunc := fmt.Sprintf("%s OVER (PARTITION BY %s ORDER BY %s)",
		column, partition, orderBy)
	qb.columns = append(qb.columns, windowFunc)
	return qb
}

// RowNumber добавляет ROW_NUMBER()
func (qb *QueryBuilder) RowNumber(partition string, orderBy string, alias string) *QueryBuilder {
	windowFunc := fmt.Sprintf("ROW_NUMBER() OVER (PARTITION BY %s ORDER BY %s) AS %s",
		partition, orderBy, alias)
	qb.columns = append(qb.columns, windowFunc)
	return qb
}

// Rank добавляет RANK()
func (qb *QueryBuilder) Rank(partition string, orderBy string, alias string) *QueryBuilder {
	windowFunc := fmt.Sprintf("RANK() OVER (PARTITION BY %s ORDER BY %s) AS %s",
		partition, orderBy, alias)
	qb.columns = append(qb.columns, windowFunc)
	return qb
}

// DenseRank добавляет DENSE_RANK()
func (qb *QueryBuilder) DenseRank(partition string, orderBy string, alias string) *QueryBuilder {
	windowFunc := fmt.Sprintf("DENSE_RANK() OVER (PARTITION BY %s ORDER BY %s) AS %s",
		partition, orderBy, alias)
	qb.columns = append(qb.columns, windowFunc)
	return qb
}

// WhereRaw добавляет сырое условие WHERE
func (qb *QueryBuilder) WhereRaw(sql string, args ...interface{}) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   sql,
		args:     args,
	})
	return qb
}

// OrWhereRaw добавляет сырое условие через OR
func (qb *QueryBuilder) OrWhereRaw(sql string, args ...interface{}) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "OR",
		clause:   sql,
		args:     args,
	})
	return qb
}

// WhereDate добавляет условие по дате
func (qb *QueryBuilder) WhereDate(column string, operator string, value time.Time) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("DATE(%s) %s ?", column, operator),
		args:     []interface{}{value},
	})
	return qb
}

// WhereBetweenDates добавляет условие между датами
func (qb *QueryBuilder) WhereBetweenDates(column string, start time.Time, end time.Time) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("DATE(%s) BETWEEN ? AND ?", column),
		args:     []interface{}{start.Format("2006-01-02"), end.Format("2006-01-02")},
	})
	return qb
}

// WhereDateTime добавляет условие по дате и времени
func (qb *QueryBuilder) WhereDateTime(column string, operator string, value time.Time) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("%s %s ?", column, operator),
		args:     []interface{}{value.Format("2006-01-02 15:04:05")},
	})
	return qb
}

// WhereBetweenDateTime добавляет условие между датами и временем
func (qb *QueryBuilder) WhereBetweenDateTime(column string, start time.Time, end time.Time) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("%s BETWEEN ? AND ?", column),
		args: []interface{}{
			start.Format("2006-01-02 15:04:05"),
			end.Format("2006-01-02 15:04:05"),
		},
	})
	return qb
}

// WhereYear добавляет условие по году
func (qb *QueryBuilder) WhereYear(column string, operator string, year int) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("EXTRACT(YEAR FROM %s) %s ?", column, operator),
		args:     []interface{}{year},
	})
	return qb
}

// WhereMonth добавляет условие по месяцу
func (qb *QueryBuilder) WhereMonth(column string, operator string, month int) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("EXTRACT(MONTH FROM %s) %s ?", column, operator),
		args:     []interface{}{month},
	})
	return qb
}

// WhereDay добавляет условие по дню
func (qb *QueryBuilder) WhereDay(column string, operator string, day int) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("EXTRACT(DAY FROM %s) %s ?", column, operator),
		args:     []interface{}{day},
	})
	return qb
}

// WhereTime добавляет условие по времени (без учета даты)
func (qb *QueryBuilder) WhereTime(column string, operator string, value time.Time) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("TIME(%s) %s ?", column, operator),
		args:     []interface{}{value.Format("15:04:05")},
	})
	return qb
}

// WhereDateIsNull проверяет является ли дата NULL
func (qb *QueryBuilder) WhereDateIsNull(column string) *QueryBuilder {
	return qb.WhereNull(column)
}

// WhereDateIsNotNull проверяет является ли дата NOT NULL
func (qb *QueryBuilder) WhereDateIsNotNull(column string) *QueryBuilder {
	return qb.WhereNotNull(column)
}

// WhereCurrentDate добавляет условие на текущую дату
func (qb *QueryBuilder) WhereCurrentDate(column string, operator string) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("DATE(%s) %s CURRENT_DATE", column, operator),
	})
	return qb
}

// WhereLastDays добавляет условие за последние n дней
func (qb *QueryBuilder) WhereLastDays(column string, days int) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("DATE(%s) >= CURRENT_DATE - INTERVAL ? DAY", column),
		args:     []interface{}{days},
	})
	return qb
}

// WhereWeekday добавляет условие по дню недели (1 = Понедельник, 7 = Воскресенье)
func (qb *QueryBuilder) WhereWeekday(column string, operator string, weekday int) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("EXTRACT(DOW FROM %s) %s ?", column, operator),
		args:     []interface{}{weekday},
	})
	return qb
}

// WhereQuarter добавляет условие по кварталу (1-4)
func (qb *QueryBuilder) WhereQuarter(column string, operator string, quarter int) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("EXTRACT(QUARTER FROM %s) %s ?", column, operator),
		args:     []interface{}{quarter},
	})
	return qb
}

// WhereWeek добавляет условие по номеру недели в году (1-53)
func (qb *QueryBuilder) WhereWeek(column string, operator string, week int) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("EXTRACT(WEEK FROM %s) %s ?", column, operator),
		args:     []interface{}{week},
	})
	return qb
}

// WhereDateRange добавляет условие по диапазону дат с включением/исключением границ
func (qb *QueryBuilder) WhereDateRange(column string, start time.Time, end time.Time, inclusive bool) *QueryBuilder {
	if inclusive {
		return qb.WhereBetweenDates(column, start, end)
	}

	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("DATE(%s) > ? AND DATE(%s) < ?", column, column),
		args:     []interface{}{start.Format("2006-01-02"), end.Format("2006-01-02")},
	})
	return qb
}

// WhereNextDays добавляет условие на следующие n дней
func (qb *QueryBuilder) WhereNextDays(column string, days int) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("DATE(%s) <= CURRENT_DATE + INTERVAL ? DAY AND DATE(%s) >= CURRENT_DATE", column, column),
		args:     []interface{}{days},
	})
	return qb
}

// WhereDateBetweenColumns проверяет, что дата находится между значениями двух других колонок
func (qb *QueryBuilder) WhereDateBetweenColumns(dateColumn string, startColumn string, endColumn string) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("DATE(%s) BETWEEN DATE(%s) AND DATE(%s)", dateColumn, startColumn, endColumn),
	})
	return qb
}

// WhereAge добавляет условие по возрасту (для дат рождения)
func (qb *QueryBuilder) WhereAge(column string, operator string, age int) *QueryBuilder {
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf("EXTRACT(YEAR FROM AGE(%s)) %s ?", column, operator),
		args:     []interface{}{age},
	})
	return qb
}

// getDateFunctions возвращает функции для текущей СУБД
func (qb *QueryBuilder) getDateFunctions() DateFunctions {
	if qb.getDriverName() == "postgres" {
		return DateFunctions{
			DateDiff:    "DATE_PART('day', %s::timestamp - %s::timestamp)",
			DateTrunc:   "DATE_TRUNC",
			DateFormat:  "TO_CHAR",
			TimeZone:    "AT TIME ZONE",
			Extract:     "EXTRACT",
			DateAdd:     "% + INTERVAL '% %'",
			CurrentDate: "CURRENT_DATE",
		}
	}
	return DateFunctions{
		DateDiff:    "DATEDIFF(%s, %s)",
		DateTrunc:   "DATE_FORMAT", // MySQL не имеет прямого аналога DATE_TRUNC
		DateFormat:  "DATE_FORMAT",
		TimeZone:    "CONVERT_TZ",
		Extract:     "EXTRACT",
		DateAdd:     "DATE_ADD(%, INTERVAL % %)",
		CurrentDate: "CURDATE()",
	}
}

// WhereDateDiff добавляет условие по разнице между датами
func (qb *QueryBuilder) WhereDateDiff(column1 string, column2 string, operator string, days int) *QueryBuilder {
	df := qb.getDateFunctions()
	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   fmt.Sprintf(df.DateDiff+" %s ?", column1, column2, operator),
		args:     []interface{}{days},
	})
	return qb
}

// WhereTimeOverlap проверяет пересечение временных интервалов
func (qb *QueryBuilder) WhereTimeOverlap(startCol1, endCol1, startCol2, endCol2 string) *QueryBuilder {
	if qb.getDriverName() == "postgres" {
		qb.conditions = append(qb.conditions, Condition{
			operator: "AND",
			clause:   fmt.Sprintf("(%s, %s) OVERLAPS (%s, %s)", startCol1, endCol1, startCol2, endCol2),
		})
	} else {
		// MySQL не имеет OVERLAPS - используем логическое выражение
		qb.conditions = append(qb.conditions, Condition{
			operator: "AND",
			clause:   fmt.Sprintf("(%s <= %s AND %s >= %s)", startCol1, endCol2, endCol1, startCol2),
		})
	}
	return qb
}

// WhereDateTrunc добавляет условие с усечением даты
func (qb *QueryBuilder) WhereDateTrunc(part string, column string, operator string, value time.Time) *QueryBuilder {
	df := qb.getDateFunctions()
	var clause string
	var args []interface{}

	if qb.getDriverName() == "postgres" {
		clause = fmt.Sprintf("%s(?, %s) %s ?", df.DateTrunc, column, operator)
		args = []interface{}{part, value}
	} else {
		// Преобразуем part в формат MySQL
		format := getMySQLDateFormat(part)
		clause = fmt.Sprintf("%s(%s, ?) %s ?", df.DateTrunc, column, operator)
		args = []interface{}{format, value.Format(format)}
	}

	qb.conditions = append(qb.conditions, Condition{
		operator: "AND",
		clause:   clause,
		args:     args,
	})
	return qb
}

// getMySQLDateFormat преобразует части даты в формат MySQL
func getMySQLDateFormat(part string) string {
	switch strings.ToLower(part) {
	case "year":
		return "%Y"
	case "month":
		return "%Y-%m"
	case "day":
		return "%Y-%m-%d"
	case "hour":
		return "%Y-%m-%d %H"
	case "minute":
		return "%Y-%m-%d %H:%i"
	default:
		return "%Y-%m-%d %H:%i:%s"
	}
}

// WhereTimeWindow добавляет условие попадания времени в окно
func (qb *QueryBuilder) WhereTimeWindow(column string, startTime, endTime time.Time) *QueryBuilder {
	if qb.getDriverName() == "postgres" {
		qb.conditions = append(qb.conditions, Condition{
			operator: "AND",
			clause:   fmt.Sprintf("EXTRACT(HOUR FROM %s) * 60 + EXTRACT(MINUTE FROM %s) BETWEEN ? AND ?", column, column),
			args: []interface{}{
				startTime.Hour()*60 + startTime.Minute(),
				endTime.Hour()*60 + endTime.Minute(),
			},
		})
	} else {
		qb.conditions = append(qb.conditions, Condition{
			operator: "AND",
			clause:   fmt.Sprintf("TIME(%s) BETWEEN ? AND ?", column),
			args: []interface{}{
				startTime.Format("15:04:05"),
				endTime.Format("15:04:05"),
			},
		})
	}
	return qb
}

// WhereBusinessDays добавляет условие только по рабочим дням
func (qb *QueryBuilder) WhereBusinessDays(column string) *QueryBuilder {
	if qb.getDriverName() == "postgres" {
		qb.conditions = append(qb.conditions, Condition{
			operator: "AND",
			clause:   fmt.Sprintf("EXTRACT(DOW FROM %s) BETWEEN 1 AND 5", column),
		})
	} else {
		qb.conditions = append(qb.conditions, Condition{
			operator: "AND",
			clause:   fmt.Sprintf("WEEKDAY(%s) < 5", column),
		})
	}
	return qb
}

// WhereDateFormat добавляет условие по отформатированной дате
func (qb *QueryBuilder) WhereDateFormat(column string, format string, operator string, value string) *QueryBuilder {
	df := qb.getDateFunctions()

	if qb.getDriverName() == "postgres" {
		// Преобразуем формат из MySQL в PostgreSQL
		pgFormat := convertToPostgresFormat(format)
		qb.conditions = append(qb.conditions, Condition{
			operator: "AND",
			clause:   fmt.Sprintf("%s(%s, ?) %s ?", df.DateFormat, column, operator),
			args:     []interface{}{pgFormat, value},
		})
	} else {
		qb.conditions = append(qb.conditions, Condition{
			operator: "AND",
			clause:   fmt.Sprintf("%s(%s, ?) %s ?", df.DateFormat, column, operator),
			args:     []interface{}{format, value},
		})
	}
	return qb
}

// convertToPostgresFormat преобразует формат даты из MySQL в PostgreSQL
func convertToPostgresFormat(mysqlFormat string) string {
	replacer := strings.NewReplacer(
		"%Y", "YYYY",
		"%m", "MM",
		"%d", "DD",
		"%H", "HH24",
		"%i", "MI",
		"%s", "SS",
	)
	return replacer.Replace(mysqlFormat)
}

// WhereTimeZone добавляет условие с учетом временной зоны
func (qb *QueryBuilder) WhereTimeZone(column string, operator string, value time.Time, timezone string) *QueryBuilder {
	df := qb.getDateFunctions()

	if qb.getDriverName() == "postgres" {
		qb.conditions = append(qb.conditions, Condition{
			operator: "AND",
			clause:   fmt.Sprintf("%s %s ? %s ?", column, df.TimeZone, operator),
			args:     []interface{}{timezone, value},
		})
	} else {
		qb.conditions = append(qb.conditions, Condition{
			operator: "AND",
			clause:   fmt.Sprintf("%s(%s, 'UTC', ?) %s ?", df.TimeZone, column, operator),
			args:     []interface{}{timezone, value},
		})
	}
	return qb
}