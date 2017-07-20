// Copyright © 2017 Syndesis Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs github-example-app",
	Long:  `Runs github-example-app.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()

	},
}

func init() {
	RootCmd.AddCommand(runCmd)

	runCmd.Flags().Int("github-app-id", 0, "GitHub app ID")
	v.BindPFlag("github.appId", runCmd.Flags().Lookup("github-app-id"))
	runCmd.Flags().String("github-app-private-key", "", "GitHub app private key file")
	v.BindPFlag("github.privateKey", runCmd.Flags().Lookup("github-app-private-key"))
}
