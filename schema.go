package dblayer

import (
	"fmt"
	"strings"
)

// Schema представляет построитель схемы таблицы
type Schema struct {
	dbl         *DBLayer
	name        string
	columns     []Column
	primaryKey  []string
	uniqueKeys  map[string][]string
	indexes     map[string][]string
	foreignKeys map[string]*ForeignKey
	engine      string
	charset     string
	collate     string
	comment     string
	temporary   bool
	ifNotExists bool
	commands    []string

	mode string // "create" или "update"

}

func (s *Schema) BuildCreate() string {
	return s.dbl.schemaDialect.BuildCreateTable(s)
}
func (s *Schema) BuildAlter() string {
	return s.dbl.schemaDialect.BuildAlterTable(s)
}

// Добавляем методы для обновления
func (s *Schema) RenameColumn(from, to string) *Schema {
	if s.mode == "update" {
		s.commands = append(s.commands, fmt.Sprintf(
			"RENAME COLUMN %s TO %s",
			from, to,
		))
	}
	return s
}

// Timestamps добавляет поля created_at и updated_at
func (s *Schema) Timestamps() *Schema {
	s.Timestamp("created_at").Default("CURRENT_TIMESTAMP")
	s.Timestamp("updated_at").Nullable()
	return s
}

// Morphs добавляет поля для полиморфных отношений
func (s *Schema) Morphs(name string) *Schema {
	s.Integer(name + "_id")
	s.String(name+"_type", 255)
	s.Index("idx_"+name, name+"_id", name+"_type")
	return s
}

// UniqueIndex добавляет уникальный индекс
func (s *Schema) UniqueIndex(name string, columns ...string) *Schema {
	return s.UniqueKey(name, columns...)
}

// FullText добавляет полнотекстовый индекс
func (s *Schema) FullText(name string, columns ...string) *Schema {
	// Реализация зависит от типа БД
	if s.dbl.db.DriverName() == "mysql" {
		s.indexes[name] = columns
		return s
	}
	return s
}

// Audit добавляет поля аудита
func (s *Schema) Audit() *Schema {
	s.ForeignKey("created_by", "users", "id")
	s.ForeignKey("updated_by", "users", "id")
	s.ForeignKey("deleted_by", "users", "id")
	s.Timestamps()
	s.SoftDeletes()
	return s
}

// PrimaryKey устанавливает первичный ключ
func (s *Schema) PrimaryKey(columns ...string) *Schema {
	s.primaryKey = columns
	return s
}

// UniqueKey добавляет уникальный ключ
func (s *Schema) UniqueKey(name string, columns ...string) *Schema {
	s.uniqueKeys[name] = columns
	return s
}

// Engine устанавливает движок таблицы
func (s *Schema) Engine(engine string) *Schema {
	s.engine = engine
	return s
}

// Charset устанавливает кодировку
func (s *Schema) Charset(charset string) *Schema {
	s.charset = charset
	return s
}

// Collate устанавливает сравнение
func (s *Schema) Collate(collate string) *Schema {
	s.collate = collate
	return s
}

// Comment добавляет комментарий
func (s *Schema) Comment(comment string) *Schema {
	s.comment = comment
	return s
}

// Temporary делает таблицу временной
func (s *Schema) Temporary() *Schema {
	s.temporary = true
	return s
}

// IfNotExists добавляет проверку существования
func (s *Schema) IfNotExists() *Schema {
	s.ifNotExists = true
	return s
}

// DropColumn удаляет колонку
func (s *Schema) DropColumn(name string) *Schema {
	s.commands = append(s.commands, fmt.Sprintf("DROP COLUMN %s", name))
	return s
}

// ModifyColumn изменяет колонку
func (s *Schema) ModifyColumn(column Column) *Schema {
	s.commands = append(s.commands, fmt.Sprintf(
		"MODIFY COLUMN %s",
		buildColumnDefinition(column),
	))
	return s
}

// AddIndex добавляет индекс
func (s *Schema) AddIndex(name string, columns []string, unique bool) *Schema {
	indexType := "INDEX"
	if unique {
		indexType = "UNIQUE INDEX"
	}
	s.commands = append(s.commands, fmt.Sprintf(
		"ADD %s %s (%s)",
		indexType, name,
		strings.Join(columns, ", "),
	))
	return s
}

// DropIndex удаляет индекс
func (s *Schema) DropIndex(name string) *Schema {
	s.commands = append(s.commands, fmt.Sprintf("DROP INDEX %s", name))
	return s
}

// RenameTable переименовывает таблицу
func (s *Schema) RenameTable(newName string) *Schema {
	s.commands = append(s.commands, fmt.Sprintf("RENAME TO %s", newName))
	return s
}

// ChangeEngine меняет движок таблицы
func (s *Schema) ChangeEngine(engine string) *Schema {
	s.commands = append(s.commands, fmt.Sprintf("ENGINE = %s", engine))
	return s
}

// ChangeCharset меняет кодировку
func (s *Schema) ChangeCharset(charset, collse string) *Schema {
	s.commands = append(s.commands, fmt.Sprintf(
		"CHARACTER SET = %s COLLsE = %s",
		charset, collse,
	))
	return s
}
