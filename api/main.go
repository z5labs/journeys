package main

import (
	"bytes"
	_ "embed"

	"github.com/z5labs/humus/rest"
	"github.com/z5labs/journeys/api/app"
)

//go:embed config.yaml
var configBytes []byte

func main() {
	rest.Run(bytes.NewBuffer(configBytes), app.Init)
}
