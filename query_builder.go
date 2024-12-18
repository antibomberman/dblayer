package dblayer

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"
)

type JoinType string

const (
	InnerJoin JoinType = "INNER JOIN"
	LeftJoin  JoinType = "LEFT JOIN"
	RightJoin JoinType = "RIGHT JOIN"
	CrossJoin JoinType = "CROSS JOIN"
)

type Join struct {
	Type      JoinType
	Table     string
	Condition string
}

type Condition struct {
	operator string
	clause   string
	nested   []Condition
	args     []interface{}
}

type QueryBuilder struct {
	table      string
	conditions []Condition
	db         interface{} // может быть *sqlx.DB или *sqlx.Tx
	columns    []string
	orderBy    []string
	groupBy    []string
	having     string
	limit      int
	offset     int
	joins      []Join
	alias      string
}

// Executor интерфейс для выполнения запросов
type Executor interface {
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
	NamedExec(query string, arg interface{}) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
}

// getExecutor возвращает исполнитель запросов
func (qb *QueryBuilder) getExecutor() Executor {
	switch db := qb.db.(type) {
	case *sqlx.Tx:
		return db
	case *sqlx.DB:
		return db
	default:
		panic("invalid database executor")
	}
}

// rebindQuery преобразует плейсхолдеры под нужный диалект SQL
func (qb *QueryBuilder) rebindQuery(query string) string {
	switch db := qb.db.(type) {
	case *sqlx.DB:
		return db.Rebind(query)
	case *sqlx.Tx:
		return db.Rebind(query)
	default:
		return query
	}
}

// buildConditions собирает условия WHERE в строку
func buildConditions(conditions []Condition) string {
	var parts []string

	for i, cond := range conditions {
		var part string

		if len(cond.nested) > 0 {
			nestedSQL := buildConditions(cond.nested)
			part = "(" + nestedSQL + ")"
		} else {
			part = cond.clause
		}

		if i == 0 {
			parts = append(parts, part)
		} else {
			parts = append(parts, cond.operator+" "+part)
		}
	}

	return strings.Join(parts, " ")
}

// buildQuery собирает полный SQL запрос
func (qb *QueryBuilder) buildQuery() (string, []interface{}) {
	var args []interface{}

	selectClause := "*"
	if len(qb.columns) > 0 {
		selectClause = strings.Join(qb.columns, ", ")
	}

	tableName := qb.table
	if qb.alias != "" {
		tableName = fmt.Sprintf("%s AS %s", tableName, qb.alias)
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", selectClause, tableName)

	for _, join := range qb.joins {
		if join.Type == CrossJoin {
			sql += fmt.Sprintf(" %s %s", join.Type, join.Table)
		} else {
			sql += fmt.Sprintf(" %s %s ON %s", join.Type, join.Table, join.Condition)
		}
	}

	if len(qb.conditions) > 0 {
		whereSQL := buildConditions(qb.conditions)
		sql += " WHERE " + whereSQL

		// Собираем все аргументы из условий
		for _, cond := range qb.conditions {
			args = append(args, cond.args...)
		}
	}

	if len(qb.groupBy) > 0 {
		sql += " GROUP BY " + strings.Join(qb.groupBy, ", ")
	}

	if qb.having != "" {
		sql += " HAVING " + qb.having
	}

	if len(qb.orderBy) > 0 {
		sql += " ORDER BY " + strings.Join(qb.orderBy, ", ")
	}

	if qb.limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", qb.limit)
	}

	if qb.offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", qb.offset)
	}
	return sql, args
}
func (qb *QueryBuilder) getStructInfo(data interface{}) (fields []string, placeholders []string) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		if tag := t.Field(i).Tag.Get("db"); tag != "" && tag != "-" && tag != "id" {
			fields = append(fields, tag)
			placeholders = append(placeholders, ":"+tag)
		}
	}
	return
}
