package cfg

import (
	"fmt"
	"net/url"
	"reflect"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	ini "gopkg.in/ini.v1"
)

// ErrIdpAccountNotFound returned if the idp account is not found in the configuration file
var ErrIdpAccountNotFound = errors.New("IDP account not found, run configure to set it up")

const (
	// DefaultConfigPath the default saml2aws configuration path
	DefaultConfigPath = "~/.saml2aws"

	// DefaultAmazonWebservicesURN URN used when authenticating to aws using SAML
	// NOTE: This only needs to be changed to log into GovCloud
	DefaultAmazonWebservicesURN = "urn:amazon:webservices"

	// DefaultSessionDuration this is the default session duration which can be overridden in the AWS console
	// see https://aws.amazon.com/blogs/security/enable-federated-api-access-to-your-aws-resources-for-up-to-12-hours-using-iam-roles/
	DefaultSessionDuration = 3600

	// DefaultProfile this is the default profile name used to save the credentials in the aws cli
	DefaultProfile = "saml"
)

// IDPAccount saml IDP account
type IDPAccount struct {
	AppID                string `ini:"app_id"` // used by OneLogin
	URL                  string `ini:"url"`
	Username             string `ini:"username"`
	Provider             string `ini:"provider"`
	MFA                  string `ini:"mfa"`
	SkipVerify           bool   `ini:"skip_verify"`
	Timeout              int    `ini:"timeout"`
	AmazonWebservicesURN string `ini:"aws_urn"`
	SessionDuration      int    `ini:"aws_session_duration"`
	Profile              string `ini:"aws_profile"`
	Subdomain            string `ini:"subdomain"` // used by OneLogin
	RoleARN              string `ini:"role_arn"`
}

func (ia IDPAccount) String() string {
	var appID string
	if ia.Provider == "OneLogin" {
		appID = fmt.Sprintf(`
  AppID: %s
  Subdomain: %s`, ia.AppID, ia.Subdomain)
	}

	return fmt.Sprintf(`account {%s
  URL: %s
  Username: %s
  Provider: %s
  MFA: %s
  SkipVerify: %v
  AmazonWebservicesURN: %s
  SessionDuration: %d
  Profile: %s
  RoleARN: %s
}`, appID, ia.URL, ia.Username, ia.Provider, ia.MFA, ia.SkipVerify, ia.AmazonWebservicesURN, ia.SessionDuration, ia.Profile, ia.RoleARN)
}

// Validate validate the required / expected fields are set
func (ia *IDPAccount) Validate() error {
	if ia.Provider == "OneLogin" {
		if ia.AppID == "" {
			return errors.New("app ID empty in idp account")
		}
		if ia.Subdomain == "" {
			return errors.New("subdomain empty in idp account")
		}
	}

	if ia.URL == "" {
		return errors.New("URL empty in idp account")
	}

	_, err := url.Parse(ia.URL)
	if err != nil {
		return errors.New("URL parse failed")
	}

	if ia.Provider == "" {
		return errors.New("Provider empty in idp account")
	}

	if ia.MFA == "" {
		return errors.New("MFA empty in idp account")
	}

	if ia.Profile == "" {
		return errors.New("Profile empty in idp account")
	}

	return nil
}

// NewIDPAccount Create an idp account and fill in any default fields with sane values
func NewIDPAccount() *IDPAccount {
	return &IDPAccount{
		AmazonWebservicesURN: DefaultAmazonWebservicesURN,
		SessionDuration:      DefaultSessionDuration,
		Profile:              DefaultProfile,
	}
}

// ConfigManager manage the various IDP account settings
type ConfigManager struct {
	configPath string
}

// NewConfigManager build a new config manager and optionally override the config path
func NewConfigManager(configFile string) (*ConfigManager, error) {

	if configFile == "" {
		configFile = DefaultConfigPath
	}

	configPath, err := homedir.Expand(configFile)
	if err != nil {
		return nil, err
	}

	return &ConfigManager{configPath}, nil
}

// SaveIDPAccount save idp account
func (cm *ConfigManager) SaveIDPAccount(idpAccountName string, account *IDPAccount) error {

	if err := account.Validate(); err != nil {
		return errors.Wrap(err, "Account validation failed")
	}

	cfg, err := ini.LoadSources(ini.LoadOptions{Loose: true}, cm.configPath)
	if err != nil {
		return errors.Wrap(err, "Unable to load configuration file")
	}

	newSec, err := cfg.NewSection(idpAccountName)
	if err != nil {
		return errors.Wrap(err, "Unable to build a new section in configuration file")
	}

	err = newSec.ReflectFrom(account)
	if err != nil {
		return errors.Wrap(err, "Unable to save account to configuration file")
	}

	err = cfg.SaveTo(cm.configPath)
	if err != nil {
		return errors.Wrap(err, "Failed to save configuration file")
	}
	return nil
}

// LoadIDPAccount load the idp account and default to an empty one if it doesn't exist
func (cm *ConfigManager) LoadIDPAccount(idpAccountName string) (*IDPAccount, error) {

	cfg, err := ini.LoadSources(ini.LoadOptions{Loose: true}, cm.configPath)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to load configuration file")
	}

	return readAccount(idpAccountName, cfg)
}

// LoadVerifyIDPAccount load the idp account and verify it isn't empty
func (cm *ConfigManager) LoadVerifyIDPAccount(idpAccountName string) (*IDPAccount, error) {

	cfg, err := ini.LoadSources(ini.LoadOptions{Loose: true}, cm.configPath)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to load configuration file")
	}

	account, err := readAccount(idpAccountName, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to read idp account")
	}

	if reflect.DeepEqual(account, NewIDPAccount()) {
		return nil, ErrIdpAccountNotFound
	}

	return account, nil
}

// IsErrIdpAccountNotFound check if the error is a ErrIdpAccountNotFound
func IsErrIdpAccountNotFound(err error) bool {
	return err == ErrIdpAccountNotFound
}

func readAccount(idpAccountName string, cfg *ini.File) (*IDPAccount, error) {

	account := NewIDPAccount()

	sec := cfg.Section(idpAccountName)

	err := sec.MapTo(account)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to map account")
	}

	return account, nil
}
