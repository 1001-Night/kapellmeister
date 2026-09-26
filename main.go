package main

import (
	"os"

	"gitlab.com/1001n/kapellmeister/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
