/*
Copyright © 2022 University of Pennsylvania <support@agent>>

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
	"github.com/pennsieve/pennsieve-agent/migrations"
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	"github.com/spf13/cobra"
	"log"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Agent",
	Long:  `Initializing the agent will create a local database that is used by the agent.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, err := db.InitializeDB()
		if err != nil {
			log.Println("Driver creation failed", err.Error())
		} else {
			// Run all migrations
			migrations.Run()
		}
	},
}

func init() {
}
