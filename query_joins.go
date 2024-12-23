package dblayer

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
