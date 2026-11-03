// Command wandcheck is a native browser and trainer panel for the games the
// WeMod client supports.
package main

import (
	"log"

	"github.com/yafit/nova/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}