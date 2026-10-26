package snowflake

import (
	_ "github.com/snowflakedb/gosnowflake/v2" // register database/sql driver name "snowflake"
	"github.com/supersuit-tech/permission-slip/connectors"
)

func init() {
	connectors.RegisterBuiltIn(New())
}
