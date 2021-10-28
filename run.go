package main

import (
	"github.com/dsorm/sitemapwalk/app"
	"github.com/dsorm/sitemapwalk/cmd"
)

func main() {
	app.Debug = false
	cmd.Execute()
}
