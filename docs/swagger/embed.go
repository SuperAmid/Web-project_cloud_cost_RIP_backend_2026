// Package swagger встраивает спецификацию OpenAPI, которую генерирует swag:
//
//	swag init -g main.go -o docs/swagger --outputTypes json,yaml
package swagger

import _ "embed"

//go:embed swagger.json
var JSON []byte
