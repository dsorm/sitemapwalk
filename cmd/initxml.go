/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	_ "embed"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

//go:embed sitemaps.tmpl.xml
var sitemapsTmpl []byte

// initxmlCmd represents the initxml command
var initxmlCmd = &cobra.Command{
	Use:   "initxml",
	Short: "Creates a new XML sitemap template",
	Long: `Creates a new XML sitemap template, so you can easily
start writing your root sitemap.

Usage:
sitemapwalk initxml (--filename[-f] mynewsitemap.xml)
`,
	Run: func(cmd *cobra.Command, args []string) {
		filename := "sitemaps.xml"
		str, err := cmd.Flags().GetString("filename")
		if err == nil {
			filename = str
		}
		_, err = os.Open(filename)
		if err == nil {
			fmt.Println("Error: the file already exists!")
			return
		}

		err = os.WriteFile(filename, sitemapsTmpl, 0644)
		if err != nil {
			fmt.Println("Error while writing file:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initxmlCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initxmlCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initxmlCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	initxmlCmd.Flags().StringP("filename", "f", "sitemaps.xml", "Filename of the new XML file, defaults to 'sitemaps.xml', if not specified")
}
