package aws

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	log "github.com/sirupsen/logrus"
)

const (
	ProfileString     string = "AWS_PROFILE"
	DefaultAssumeRole string = "terraform"
)

// SetLogLevel sets the logging level for the application (debug, info, warn, error).
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

// NewConfigurationManager creates a new ConfigurationManager using environment and AWS credentials.
func NewConfigurationManager() (*ConfigurationManager, error) {
	return NewConfigurationManagerForRegionsAndAccounts(make([]string, 0), make([]string, 0), "")
}

// NewConfigurationManagerForRegionsAndAccounts creates a new ConfigurationManager with specified regions, accounts, and role for cross-account operations.
func NewConfigurationManagerForRegionsAndAccounts(regions []string, accounts []string, role string) (*ConfigurationManager, error) {
	cm := &ConfigurationManager{
		regions:  regions,
		accounts: accounts,
		role:     role,
	}

	log.Debug("Setting defaults")
	// Ensure shared config is considered (helps with SSO profiles when user forgot to export AWS_SDK_LOAD_CONFIG=1)
	if os.Getenv("AWS_SDK_LOAD_CONFIG") == "" {
		_ = os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
		log.Debug("AWS_SDK_LOAD_CONFIG not set; defaulting to 1 to enable shared config & SSO support")
	}

	profileFromEnv := os.Getenv("AWS_PROFILE")
	conf, err := config.LoadDefaultConfig(context.TODO(), func(o *config.LoadOptions) error {
		// If a profile is set, use it explicitly
		if profileFromEnv != "" {
			o.SharedConfigProfile = profileFromEnv
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed loading default AWS config (profile=%s): %w", profileFromEnv, err)
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
	cm.defaultProfile = profileFromEnv
	cm.defaultRegion = conf.Region

	// Attempt credential retrieval before making STS call for clearer diagnostics
	creds, credErr := conf.Credentials.Retrieve(context.TODO())
	if credErr != nil {
		return nil, fmt.Errorf("unable to retrieve AWS credentials for profile '%s' region '%s': %v. Hints: ensure 'aws sso login --profile %s' was executed, or export static credentials. If using SSO, confirm your AWS CLI v2 cache exists in ~/.aws/sso/cache and AWS_SDK_LOAD_CONFIG=1", cm.defaultProfile, conf.Region, credErr, cm.defaultProfile)
	}
	log.WithFields(log.Fields{"credential_source": creds.Source, "access_key_present": creds.AccessKeyID != "", "using_profile": profileFromEnv}).Debug("Retrieved AWS credentials")

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
			baseMsg += "\nHint: STS endpoint resolution failed. This almost always means the region was empty or invalid, or the service model is outdated. Try updating dependencies (go get -u github.com/aws/aws-sdk-go-v2/...) and passing --region explicitly."
		}
		baseMsg += "\n" + buildCredentialHint().Error()
		return nil, errors.New(baseMsg)
	}

	cm.defaultAccountID = defaultAccountID.Account

	// Enhanced cross-account role handling (in-place patch)
	// If accounts are specified and role is empty, attempt environment fallback then default constant
	if len(cm.accounts) > 0 && strings.TrimSpace(cm.role) == "" {
		if envRole := os.Getenv("AWS_AMI_MANAGER_ROLE"); strings.TrimSpace(envRole) != "" {
			cm.role = strings.TrimSpace(envRole)
			log.Debugf("Role not provided via flag; using AWS_AMI_MANAGER_ROLE env value '%s'", cm.role)
		} else {
			cm.role = "terraform" // default fallback consistent with copy/remove command
			log.Debugf("Role not provided; defaulting to '%s'", cm.role)
		}
	}

	cm.configsPerAccount = make(map[string]awsv2.Config)
	for _, account := range cm.accounts {
		account = strings.TrimSpace(account)
		if account == "" {
			log.Warn("Skipping empty account entry in --accounts / accounts list")
			continue
		}
		// Skip assuming into the default (current) account
		if cm.defaultAccountID != nil && account == *cm.defaultAccountID {
			log.WithField("account", account).Debug("Skipping assume-role for default account")
			continue
		}

		if strings.TrimSpace(cm.role) == "" {
			return nil, fmt.Errorf("cannot assume role into account %s: role name is empty after fallback attempts; pass --role or set AWS_AMI_MANAGER_ROLE", account)
		}

		confCopy := cm.defaultConfig.Copy()
		assumeArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, cm.role)
		log.WithFields(log.Fields{"account": account, "role": cm.role, "assume_role_arn": assumeArn}).Debug("Configuring assume role provider")
		confCopy.Credentials = stscreds.NewAssumeRoleProvider(sts.NewFromConfig(cm.defaultConfig), assumeArn)
		cm.configsPerAccount[account] = confCopy
	}

	log.WithFields(log.Fields{"default_account": func() string {
		if cm.defaultAccountID != nil {
			return *cm.defaultAccountID
		}
		return "<nil>"
	}(), "target_accounts": cm.accounts, "role": cm.role}).Debug("Initialized ConfigurationManager")

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
	return fmt.Errorf("%s", msg)
}

// GetDefaultRegion returns the default AWS region.
func (cm *ConfigurationManager) GetDefaultRegion() string {
	return cm.defaultRegion
}

// GetDefaultAccountID returns the default AWS account ID.
func (cm *ConfigurationManager) GetDefaultAccountID() *string {
	return cm.defaultAccountID
}

// GetConfigurationForDefaultAccount returns the AWS config for the default account.
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

func (cm *ConfigurationManager) getConfigurationForAccountAndRegion(account string, region string) awsv2.Config {
	log.Debugf("getConfigurationForAccountAndRegion - Account: %s, Region: %s", account, region)
	conf := cm.getConfigurationForAccount(account)
	conf.Region = region

	return conf
}

func (cm *ConfigurationManager) getAccounts() []string {
	return cm.accounts
}

// AssumeDefaultAccountRole assumes an IAM role in the specified account and updates the manager's default configuration.
func (cm *ConfigurationManager) AssumeDefaultAccountRole(account string, role string) error {
	// Build new assumed role config based on current default
	base := cm.GetConfigurationForDefaultAccount()
	stsSvc := sts.NewFromConfig(base)
	provider := stscreds.NewAssumeRoleProvider(stsSvc, fmt.Sprintf("arn:aws:iam::%s:role/%s", account, role))
	assumed := base
	assumed.Credentials = awsv2.NewCredentialsCache(provider)

	// Verify identity
	stsAssumed := sts.NewFromConfig(assumed)
	id, err := stsAssumed.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to assume role %s in account %s: %w", role, account, err)
	}

	cm.defaultConfig = assumed
	cm.defaultAccountID = id.Account
	cm.role = role
	return nil
}
