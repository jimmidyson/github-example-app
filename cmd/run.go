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
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	"github.com/jimmidyson/github-example-app/pkg/github/apps"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs github-example-app",
	Long:  `Runs github-example-app.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()

		// Get a github client
		githubClient, err := apps.APIClientFromKeyFile(botConfig.GitHubApp.AppID, botConfig.GitHubApp.InstallationID, botConfig.GitHubApp.PrivateKeyFile)
		if err != nil {
			logger.Fatal("Failed to create github client", zap.Error(err))
		}

		// Create a github repo
		githubRepo, _, err := githubClient.Repositories.Create(context.Background(), "purehub", &github.Repository{
			Name: github.String("foo"),
		})
		if err != nil {
			logger.Fatal("Failed to create github repo", zap.Error(err))
		}

		// Configure git http client
		gitTransport, err := apps.NewGitTransportFromKeyFile(&http.Transport{}, botConfig.GitHubApp.AppID, botConfig.GitHubApp.InstallationID, botConfig.GitHubApp.PrivateKeyFile)
		if err != nil {
			logger.Fatal("Failed to create git transport", zap.Error(err))
		}
		gitHttpClient := githttp.NewClient(&http.Client{Transport: gitTransport})

		// Initialize an empty repository
		repo, err := git.PlainInit("/tmp/git", false)
		if err != nil {
			logger.Fatal("Failed to init git repo", zap.Error(err))
		}

		// Add a remote to the github repo we created above
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URL:  githubRepo.GetCloneURL(),
		})
		if err != nil {
			logger.Fatal("Failed to add remote", zap.Error(err))
		}

		// Create a dummy file
		ioutil.WriteFile("/tmp/git/testing", []byte("this is a test"), 0644)

		// Get the working tree
		wt, err := repo.Worktree()
		if err != nil {
			logger.Fatal("Failed to get worktree", zap.Error(err))
		}
		// Add the new file
		_, err = wt.Add("testing")
		if err != nil {
			logger.Fatal("Failed to add new file", zap.Error(err))
		}

		// git commit
		_, err = wt.Commit("New file testing", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Jimmi Dyson",
				Email: "jimmidyson@gmail.com",
				When:  time.Now(),
			},
		})
		if err != nil {
			logger.Fatal("Failed to commit file", zap.Error(err))
		}

		// git push
		err = repo.Push(&git.PushOptions{
			Clients: map[string]transport.Transport{
				"http":  gitHttpClient,
				"https": gitHttpClient,
			},
		})
		if err != nil {
			logger.Fatal("Failed to commit file", zap.Error(err))
		}
	},
}

func init() {
	RootCmd.AddCommand(runCmd)

	runCmd.Flags().Int("github-app-id", 0, "GitHub app ID")
	v.BindPFlag("github.appId", runCmd.Flags().Lookup("github-app-id"))
	runCmd.Flags().String("github-app-private-key", "", "GitHub app private key file")
	v.BindPFlag("github.privateKey", runCmd.Flags().Lookup("github-app-private-key"))
	runCmd.Flags().Int("github-app-installation-id", 0, "GitHub app installation id")
	v.BindPFlag("github.installationId", runCmd.Flags().Lookup("github-app-installation-id"))
}
