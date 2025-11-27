package main

import (
	"bytes"
	_ "embed"

	"github.com/z5labs/humus/rest"
	"github.com/z5labs/journeys/services/poc/app"
)

//go:embed config.yaml
var configBytes []byte

func main() {
	rest.Run(bytes.NewReader(configBytes), app.Init)
}
