package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	fmt.Printf("testapp %s\n", version)
	os.Exit(0)
}
