package snowflake

import (
	_ "github.com/snowflakedb/gosnowflake" // register database/sql driver name "snowflake"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func init() {
	connectors.RegisterBuiltIn(New())
}
