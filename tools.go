//go:build tools
// +build tools

package tools

//go:generate go run github.com/99designs/gqlgen generate

import (
	_ "github.com/vektah/gqlparser/v2"
)
