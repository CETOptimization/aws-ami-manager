package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/cloudnatives/aws-ami-manager/aws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Show resolved AWS configuration and attempt STS identity call",
	Run: func(cmd *cobra.Command, args []string) {
		runDiagnose()
	},
}

func runDiagnose() {
	start := time.Now()
	fmt.Println("== aws-ami-manager diagnostics ==")
	fmt.Printf("AWS_REGION env: %s\n", os.Getenv("AWS_REGION"))
	fmt.Printf("AWS_DEFAULT_REGION env: %s\n", os.Getenv("AWS_DEFAULT_REGION"))
	fmt.Printf("AWS_PROFILE env: %s\n", os.Getenv("AWS_PROFILE"))
	fmt.Printf("Has AWS_ACCESS_KEY_ID: %v\n", os.Getenv("AWS_ACCESS_KEY_ID") != "")
	fmt.Printf("Has AWS_SESSION_TOKEN: %v\n", os.Getenv("AWS_SESSION_TOKEN") != "")

	cm, err := aws.NewConfigurationManager()
	if err != nil {
		fmt.Println("Configuration error:")
		fmt.Println(err.Error())
		fmt.Println("You can re-run with --loglevel=debug for more detail.")
		return
	}

	fmt.Printf("Resolved Region: %s\n", cm.GetDefaultRegion())
	if acct := cm.GetDefaultAccountID(); acct != nil {
		fmt.Printf("Resolved Account ID: %s\n", *acct)
	} else {
		fmt.Println("Resolved Account ID: <nil>")
	}
	fmt.Printf("Elapsed: %s\n", time.Since(start))
	log.Info("Diagnostics complete")
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)
}
