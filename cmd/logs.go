/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

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
	"github.com/perun-cloud-inc/perunctl/services"
	"github.com/spf13/cobra"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Get logs from a container",
	Run: func(cmd *cobra.Command, args []string) {
		containerId, _ := cmd.Flags().GetString("id")
		err := services.GetLogs(containerId)
		if err != nil {
			cobra.CheckErr(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(logsCmd)
	logsCmd.Flags().StringP("id", "i", "", "Container id")

	// TODO: Implement tail flag, and re-enable it
	//logsCmd.Flags().BoolP("tail", "t", false, "Tail the logs")
}
