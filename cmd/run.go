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
	"context"
	"fmt"
	"github.com/dsorm/sitemapwalk/app"
	"github.com/jackc/pgx/v4"
	"github.com/spf13/cobra"
	"log"
	"os"
	"time"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Takes sitemap, expands it and saves it",
	Long: `Takes sitemap, expands it and saves it.
Currently only supports XML files as input and outputs only to Postgres.

Example:
sitemapwalk run -i mysitemap.xml -o postgres --execute-sql mysql.sql --dsn "user=myname password=mysecretpassword host=postgres.example.com dbname=mydatabase port=5432"
`,
	Run: func(cmd *cobra.Command, args []string) {

		ctx, ctxCancel := context.WithCancel(context.Background())
		defer ctxCancel()
		// get, check flags and init things

		// input
		input, err := cmd.Flags().GetString("input")
		if err != nil {
			log.Fatalf("Error: input flag invalid: %v\n", err.Error())
		}
		inputBytes, err := os.ReadFile(input)
		if err != nil {
			log.Fatalf("Error while reading input file: %v\n", err.Error())
		}

		// output
		outputType, err := cmd.Flags().GetString("output-type")
		if err != nil {
			log.Fatalf("Error: output-type flag invalid: %v\n", err.Error())
		} else if outputType != "postgres" {
			log.Fatalf("Error: only postgres is currently supported as output-type\n")
		}

		var conn *pgx.Conn
		if outputType == "postgres" {
			dsn, err := cmd.Flags().GetString("dsn")
			if err != nil {
				log.Fatalf("Error: dsn flag invalid: %v\n", err.Error())
			}
			conn, err = pgx.Connect(ctx, dsn)
			if err != nil {
				log.Fatalf("Error while connecting to postgres: %v\n", err.Error())
			}

			executeSql, err := cmd.Flags().GetString("execute-sql")
			if err != nil {
				log.Fatalf("Error: output-type flag invalid: %v\n", err.Error())
			}
			sqlBytes, err := os.ReadFile(executeSql)
			if err != nil {
				log.Fatalf("Error while reading sql file: %v\n", err.Error())
			}
			sql := string(sqlBytes)
			_, err = conn.Prepare(ctx, "send", sql)
			if err != nil {
				log.Fatalf("Error while parsing sql into prepared statement: %v\n", err.Error())
			}
		}

		// do work

		rootNode, err := app.LoadAndExpandSitemap(inputBytes)
		if err != nil {
			log.Fatalf("Error while loading and expanding sitemap: %v\n", err.Error())
		}

		// only postgres can be used, so do data transfer to postgres
		fmt.Println("Sending data to postgres...")
		nodeChan := make(chan app.Node, 16)
		done := make(chan bool, 1)
		nodesTransferred := uint64(0)
		go func() {
			rootNode.SendForEachUrl(nodeChan)
			done <- true
		}()

		// a fancy display for showing progress
		ctxDisplay, ctxDisplayCancel := context.WithCancel(ctx)
		cn := app.Node{}
		defer ctxDisplayCancel()
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			for {
				select {
				case <-ctxDisplay.Done():
					fmt.Printf("%v ; DONE ; %v nodes TOTAL transferred to postgres, latest loc %v\n", time.Now().String(), nodesTransferred, cn.Loc)
					return
				case <-ticker.C:
					fmt.Printf("%v ; %v nodes transferred to postgres, latest loc %v\n", time.Now().String(), nodesTransferred, cn.Loc)
				}
			}
		}()

		for {
			select {
			case cn = <-nodeChan:
				_, err = conn.Exec(ctx, "send", cn.Loc)
				if err != nil {
					log.Printf("Error while executing sql: %v\n", err.Error())
				} else {
					nodesTransferred++
				}
			case <-done:
				goto endloop
			}
		}
	endloop:
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
	runCmd.Flags().String("execute-sql", "save.sql", "sql to execute in the database, save.sql by default")
	runCmd.Flags().String("dsn", "", "DSN or URL to connect to postgres, by default looks for PGHOST, PGPORT, PGDATABASE, PGUSER, PGPASWORD environment variables and creates DSN automatically")

}
