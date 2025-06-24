package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/spf13/cobra"
	"mime"
	"sproxy/cmd"
)

func init() {
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".mjs", "application/javascript")
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".json", "application/json")
}

func main() {
	cobra.CheckErr(cmd.Execute())
}
