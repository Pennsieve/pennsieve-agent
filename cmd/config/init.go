/*
Copyright © 2022 University of Pennsylvania <support@pennsieve.io>>

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
package config

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pennsieve/pennsieve-agent/config"
	"github.com/pennsieve/pennsieve-agent/migrations"
	"github.com/spf13/cobra"
	"log"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Agent",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("init called")
		_, err := config.InitializeDB()
		if err != nil {
			log.Println("Driver creation failed", err.Error())
		} else {
			// Run all migrations
			migrations.Run()

		}
	},
}

func init() {
	//cmd.rootCmd.AddCommand(whoamiCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// whoamiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// whoamiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
