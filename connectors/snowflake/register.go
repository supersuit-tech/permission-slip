package snowflake

import (
	_ "github.com/snowflakedb/gosnowflake" // register database/sql driver name "snowflake"
	"github.com/supersuit-tech/permission-slip/connectors"
)

func init() {
	connectors.RegisterBuiltIn(New())
}
