// Copyright © 2019 Jeroen Schepens <jeroen@cloudnatives.be>
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
	"fmt"
	"os"

	"github.com/cloudnatives/aws-ami-manager/aws"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var (
	logLevel       string
	amiID          string
	regions        []string
	accounts       []string
	role           string
	regionOverride string
	profileName    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "aws-ami-manager",
	Short: "Manage copying, cleanup, and removal of AMIs across regions and accounts",
	Long: `aws-ami-manager helps you copy AMIs across multiple AWS regions and accounts, 
set launch permissions, tag them, and clean up older versions.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if regionOverride != "" {
			os.Setenv("AWS_REGION", regionOverride)
		}
		if profileName != "" {
			os.Setenv("AWS_PROFILE", profileName)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	aws.SetLogLevel(logLevel)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "loglevel", logrus.DebugLevel.String(), "Set the log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&regionOverride, "region", "", "AWS region to use (overrides AWS_REGION/AWS_DEFAULT_REGION env vars)")
	rootCmd.PersistentFlags().StringVar(&profileName, "profile", "", "AWS profile name to use (sets AWS_PROFILE before loading config)")
}
