package main

import (
	"fmt"
	"os"

	"github.com/alesr/tidskott-pi/cmd/tidskott-pi/app"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
