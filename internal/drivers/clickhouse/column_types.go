package clickhouse

// clickhouseColumnTypes are the ALTER types offered in the structure editor.
// The first entry is the type used for a new column. The catalog type of an
// existing column is shown even when it is not in this list.
var clickhouseColumnTypes = []string{
	"String",
	"FixedString(16)",
	"Bool",
	"Int8",
	"Int16",
	"Int32",
	"Int64",
	"Int128",
	"Int256",
	"UInt8",
	"UInt16",
	"UInt32",
	"UInt64",
	"UInt128",
	"UInt256",
	"Float32",
	"Float64",
	"Decimal(18,4)",
	"Date",
	"Date32",
	"DateTime",
	"DateTime64(3)",
	"UUID",
	"IPv4",
	"IPv6",
	"JSON",
	"Array(String)",
	"Map(String, String)",
	"Tuple(String, Int32)",
	"LowCardinality(String)",
	"Enum8('a' = 1, 'b' = 2)",
}

// ColumnTypes returns a copy of the ClickHouse column type list.
func (d *Driver) ColumnTypes() []string {
	out := make([]string, len(clickhouseColumnTypes))
	copy(out, clickhouseColumnTypes)
	return out
}
