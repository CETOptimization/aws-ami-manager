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

	"github.com/cloudnatives/aws-ami-manager/aws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	removeDryRun bool
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Removes an AMI in your current region",
	Long: `Removes an AMI in your current region.

E.g. ./aws-ami-manager remove --amiID=ami-075d87a3d4512bee5 --region=eu-west-1
You can target another account by adding --accounts <id> --role <RoleName>.
Use --dry-run to preview what would be deleted (AMI + snapshots).`,
	Run: func(cmd *cobra.Command, args []string) {
		runRemove()
	},
}

func runRemove() {
	cm, err := aws.NewConfigurationManager()
	if err != nil {
		log.Fatalf("Failed to initialize AWS configuration: %v", err)
	}

	// If user provided an account (first element) and role, assume into it
	if len(accounts) > 0 {
		if role == "" {
			log.Fatalf("--accounts provided but --role is empty; please specify --role <RoleName>")
		}
		if len(accounts) > 1 {
			log.Warnf("Multiple accounts provided for remove; only the first (%s) will be used", accounts[0])
		}
		acct := accounts[0]
		log.Infof("Assuming role %s in account %s for remove operation", role, acct)
		if err := cm.AssumeDefaultAccountRole(acct, role); err != nil {
			log.Fatalf("Unable to assume role %s in account %s: %v", role, acct, err)
		}
	}

	ami := aws.NewAmi(amiID)
	ami.SourceRegion = cm.GetDefaultRegion()

	aws.ConfigManager = cm

	err = ami.RemoveAmi(removeDryRun)

	if err != nil {
		log.Fatal(err)
	}

	if removeDryRun {
		log.Infof("[dry-run] Completed successfully; no changes made for AMI %s", ami.SourceAmiID)
		return
	}
	log.Infof("AMI %s has been removed successfully", ami.SourceAmiID)
}

func init() {
	rootCmd.AddCommand(removeCmd)

	removeCmd.Flags().StringVar(&amiID, "amiID", "", "The source AMI ID, e.g. aws-0e38957fc6310ea8b")
	_ = removeCmd.MarkFlagRequired("amiID")

	removeCmd.Flags().StringSliceVar(&accounts, "accounts", []string{}, "Optional: Account ID(s) to assume into for this operation (only first is used).")
	removeCmd.Flags().StringVar(&role, "role", aws.DefaultAssumeRole, fmt.Sprintf("Role name to assume in the provided account. Defaults to '%s'. When --accounts is set this role must exist in that account.", aws.DefaultAssumeRole))
	removeCmd.Flags().BoolVar(&removeDryRun, "dry-run", false, "Show what would be removed without performing deregistration or snapshot deletion.")
}
