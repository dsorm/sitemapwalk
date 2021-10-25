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
	"fmt"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Takes sitemap, expands it and saves it",
	Long: `Takes sitemap, expands it and saves it.
Currently only supports XML files as input and outputs only to Postgres.

Example:
sitemapwalk run -i mysitemap.xml -o postgres --execute-sql mysql.sql --db "user=myname password=mysecretpassword host=postgres.example.com dbname=mydatabase port=5432"
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("run called")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	runCmd.Flags().StringP("input", "i", "sitemaps.xml", "Path to XML sitemap")
	runCmd.Flags().StringP("output-type", "o", "postgres", "where to save the output, postgres by default")
	runCmd.Flags().String("execute-sql", "postgres", "sql to execute in the database, save.sql by default")
	runCmd.Flags().String("db", "", "DSN to connect to postgres, by default looks for PGHOST, PGPORT, PGDATABASE, PGUSER, PGPASWORD environment variables and creates DSN automatically")

}
