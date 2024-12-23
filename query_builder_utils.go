package dblayer

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"
)

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
	return qb.getExecutor().Rebind(query)
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

// getStructInfo получает информацию о полях структуры
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

// getDriverName возвращает имя драйвера базы данных
func (qb *QueryBuilder) getDriverName() string {
	return qb.dbl.db.DriverName()
	// if db, ok := qb.db.(*sqlx.DB); ok {
	// 	return db.DriverName()
	// }
	// if tx, ok := qb.db.(*sqlx.Tx); ok {
	// 	return tx.DriverName()
	// }
	// return ""
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
