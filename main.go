package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/spf13/cobra"
	"sproxy/cmd"
)

func main() {
	cobra.CheckErr(cmd.Execute())
}
