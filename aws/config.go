package aws

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"os"
	"strings"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	log "github.com/sirupsen/logrus"
)

const (
	ProfileString string = "AWS_PROFILE"
)

func SetLogLevel(level string) {
	logLevel, err := log.ParseLevel(level)
	if err != nil {
		log.Fatalf("Invalid loglevel: %s", level)
	}
	log.SetLevel(logLevel)
}

type ConfigurationManager struct {
	defaultConfig    awsv2.Config
	defaultRegion    string
	defaultProfile   string
	defaultAccountID *string

	regions  []string
	accounts []string

	configsPerAccount map[string]awsv2.Config

	role string
}

func NewConfigurationManager() (*ConfigurationManager, error) {
	return NewConfigurationManagerForRegionsAndAccounts(make([]string, 0), make([]string, 0), "")
}

func NewConfigurationManagerForRegionsAndAccounts(regions []string, accounts []string, role string) (*ConfigurationManager, error) {
	cm := &ConfigurationManager{
		regions:  regions,
		accounts: accounts,
		role:     role,
	}

	log.Debug("Setting defaults")
	conf, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed loading default AWS config: %w", err)
	}

	// Ensure region is set
	if conf.Region == "" {
		if r := os.Getenv("AWS_REGION"); r != "" {
			conf.Region = r
		} else if r := os.Getenv("AWS_DEFAULT_REGION"); r != "" {
			conf.Region = r
		} else {
			conf.Region = "us-east-1" // fallback
			log.Debug("No region found in config/env; falling back to us-east-1")
		}
	}

	cm.defaultConfig = conf
	cm.defaultProfile = os.Getenv(ProfileString)
	cm.defaultRegion = conf.Region

	log.WithFields(log.Fields{
		"resolved_region":        conf.Region,
		"env_AWS_REGION":         os.Getenv("AWS_REGION"),
		"env_AWS_DEFAULT_REGION": os.Getenv("AWS_DEFAULT_REGION"),
		"profile":                cm.defaultProfile,
		"has_access_key":         os.Getenv("AWS_ACCESS_KEY_ID") != "",
		"has_session_token":      os.Getenv("AWS_SESSION_TOKEN") != "",
	}).Debug("Resolved AWS configuration inputs")

	stsService := sts.NewFromConfig(conf)
	defaultAccountID, err := stsService.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		baseMsg := fmt.Sprintf("unable to load default account identity (region=%s): %v", conf.Region, err)
		if strings.Contains(err.Error(), "ResolveEndpointV2") {
			baseMsg += "\nHint: STS endpoint resolution failed. This almost always means the region was empty or invalid. Try passing --region or setting AWS_REGION."
		}
		baseMsg += "\n" + buildCredentialHint().Error()
		return nil, errors.New(baseMsg)
	}

	cm.defaultAccountID = defaultAccountID.Account

	cm.configsPerAccount = make(map[string]awsv2.Config)
	for _, account := range cm.accounts {
		// you shouldn't assume role in your own account. We expect this user to have sufficient permissions
		if account == *cm.defaultAccountID {
			continue
		}

		confCopy := conf.Copy()

		confCopy.Credentials = stscreds.NewAssumeRoleProvider(stsService, "arn:aws:iam::"+account+":role/"+cm.role)

		cm.configsPerAccount[account] = confCopy
	}

	return cm, nil
}

func buildCredentialHint() error {
	// Build a hint error with suggestions without spamming normal output unless debug
	msg := "credential/region resolution failed. Confirm at least one provider works: \n" +
		"  1. Environment: AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY (and AWS_SESSION_TOKEN if temp)\n" +
		"  2. Shared config: export AWS_PROFILE=name (check ~/.aws/credentials)\n" +
		"  3. SSO: run 'aws sso login' for the profile then export AWS_PROFILE\n" +
		"  4. IMDS/IRSA: if in EC2/EKS ensure instance or pod role has sts:GetCallerIdentity\n" +
		"  5. Test manually: 'aws sts get-caller-identity' should succeed in same shell\n" +
		"If using a profile, also ensure the region is set in the profile or pass --region."
	return fmt.Errorf(msg)
}

func (cm *ConfigurationManager) GetDefaultRegion() string {
	return cm.defaultRegion
}

func (cm *ConfigurationManager) GetDefaultAccountID() *string {
	return cm.defaultAccountID
}

func (cm *ConfigurationManager) loadConfiguration() {
	log.Debug("Load configuration")

}

func (cm *ConfigurationManager) GetConfigurationForDefaultAccount() awsv2.Config {
	log.Debug("GetConfigurationForDefaultAccount")
	return cm.getConfigurationForAccount(*cm.defaultAccountID)
}

func (cm *ConfigurationManager) getConfigurationForAccount(account string) awsv2.Config {
	log.Debugf("getConfigurationForAccount: account: %s", account)
	if account == *cm.defaultAccountID {
		return cm.defaultConfig
	}
	return cm.configsPerAccount[account]
}

func (cm *ConfigurationManager) getConfigurationForDefaultAccountAndRegion(region string) awsv2.Config {
	log.Debugf("getConfigurationForDefaultAccountAndRegion: region: %s", region)
	config := cm.GetConfigurationForDefaultAccount()
	config.Region = region

	return config
}

func (cm *ConfigurationManager) getConfigurationForAccountAndRegion(account string, region string) awsv2.Config {
	log.Debugf("getConfigurationForAccountAndRegion - Account: %s, Region: %s", account, region)
	conf := cm.getConfigurationForAccount(account)
	conf.Region = region

	return conf
}

func (cm *ConfigurationManager) getAccounts() []string {
	return cm.accounts
}
