package mysql

// mysqlColumnTypes are the ALTER types offered in the structure editor.
// The first entry is the type used for a new column. The catalog type of an
// existing column is shown even when it is not in this list.
var mysqlColumnTypes = []string{
	"int",
	"bigint",
	"varchar(255)",
	"text",
	"datetime",
	"timestamp",
	"decimal(10,2)",
	"json",
	"blob",
}

// ColumnTypes returns a copy of the MySQL column type list.
func (d *Driver) ColumnTypes() []string {
	out := make([]string, len(mysqlColumnTypes))
	copy(out, mysqlColumnTypes)
	return out
}
