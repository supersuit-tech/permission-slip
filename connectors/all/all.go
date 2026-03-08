// Package all blank-imports every built-in connector package so that their
// init() functions run and self-register with the connectors registry.
//
// Usage in main.go:
//
//	import _ "github.com/supersuit-tech/permission-slip-web/connectors/all"
//
// To add a new connector, create its register.go with an init() that calls
// connectors.RegisterBuiltIn(New()), then add a blank import here.
package all

import (
	_ "github.com/supersuit-tech/permission-slip-web/connectors/airtable"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/amadeus"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/asana"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/aws"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/calendly"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/confluence"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/datadog"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/discord"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/docusign"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/doordash"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/expedia"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/figma"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/github"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/google"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/hubspot"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/intercom"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/jira"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/kroger"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/linear"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/linkedin"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/make"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/meta"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/microsoft"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/monday"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/mongodb"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/mysql"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/netlify"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/notion"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/pagerduty"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/plaid"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/postgres"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/protonmail"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/quickbooks"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/redis"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/salesforce"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/sendgrid"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/shopify"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/slack"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/square"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/stripe"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/trello"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/twilio"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/vercel"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/walmart"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/x"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/zapier"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/zendesk"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/zoom"
)
