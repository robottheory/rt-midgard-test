package main

// We depend on these in our `Makefile`, but not in our code.
// To keep `go mod tidy` from removing these dependencies, we create this fake tool.
import (
	_ "github.com/99designs/gqlgen/cmd"
	_ "github.com/deepmap/oapi-codegen/pkg/codegen"
)

func main() {}
