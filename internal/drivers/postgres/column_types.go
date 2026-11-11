package postgres

// postgresColumnTypes are the ALTER types offered in the structure editor.
// The first entry is the type used for a new column. The catalog type of an
// existing column is shown even when it is not in this list.
var postgresColumnTypes = []string{
	"integer",
	"bigint",
	"text",
	"boolean",
	"varchar(255)",
	"numeric(10,2)",
	"timestamp with time zone",
	"date",
	"jsonb",
	"uuid",
}

// ColumnTypes returns a copy of the PostgreSQL column type list.
func (d *Driver) ColumnTypes() []string {
	out := make([]string, len(postgresColumnTypes))
	copy(out, postgresColumnTypes)
	return out
}
