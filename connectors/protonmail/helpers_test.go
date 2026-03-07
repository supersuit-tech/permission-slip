package protonmail

import "github.com/supersuit-tech/permission-slip-web/connectors"

func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"username": "user@proton.me",
		"password": "bridge-generated-password",
	})
}

func validCredsAllFields() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"username":  "user@proton.me",
		"password":  "bridge-generated-password",
		"smtp_host": "127.0.0.1",
		"smtp_port": "1025",
		"imap_host": "127.0.0.1",
		"imap_port": "1143",
	})
}
