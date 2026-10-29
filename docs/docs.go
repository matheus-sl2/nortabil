package docs

import _ "embed"

// SwaggerYAML contém a especificação OpenAPI embutida no binário
//
//go:embed swagger.yaml
var SwaggerYAML []byte
