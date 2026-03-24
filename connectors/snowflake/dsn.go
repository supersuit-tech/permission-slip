package snowflake

import (
	"strings"

	"github.com/snowflakedb/gosnowflake"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// buildSnowflakeConfig parses the vault DSN and optional PEM private key into a
// gosnowflake.Config. When private_key_pem is set, JWT auth is used with
// PrivateKey on the struct (not embedded in a DSN string) to avoid key material
// in logged connection URLs.
func buildSnowflakeConfig(connectionString, privateKeyPEM string) (*gosnowflake.Config, error) {
	dsn := strings.TrimSpace(connectionString)
	if dsn == "" {
		return nil, &connectors.ValidationError{Message: "missing credential: connection_string"}
	}
	pkPEM := strings.TrimSpace(privateKeyPEM)

	parseDSN := dsn
	if pkPEM != "" {
		lower := strings.ToLower(dsn)
		if strings.Contains(lower, "authenticator=") {
			return nil, &connectors.ValidationError{Message: "do not set authenticator= in connection_string when using private_key_pem; use JWT-only DSN (user@account/...) and private_key_pem"}
		}
		// ParseDSN validates password before we can switch to JWT; include JWT in the DSN
		// so password is not required, then attach the key from vault (not in the string).
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		parseDSN = dsn + sep + "authenticator=SNOWFLAKE_JWT"
	}

	cfg, err := gosnowflake.ParseDSN(parseDSN)
	if err != nil {
		return nil, &connectors.ValidationError{Message: err.Error()}
	}
	if pkPEM == "" {
		return cfg, nil
	}

	pk, err := parseRSAPrivateKeyPEM(pkPEM)
	if err != nil {
		return nil, err
	}
	cfg.Authenticator = gosnowflake.AuthTypeJwt
	cfg.PrivateKey = pk
	return cfg, nil
}
