package v0

import "github.com/ignite-hq/cli/ignite/chainconfig/common"

// ConfigYaml is the user given configuration to do additional setup
// during serve.
type ConfigYaml struct {
	Accounts  []common.Account       `yaml:"accounts"`
	Validator common.Validator       `yaml:"validator"`
	Faucet    common.Faucet          `yaml:"faucet"`
	Client    common.Client          `yaml:"client"`
	Build     common.Build           `yaml:"build"`
	Init      common.Init            `yaml:"init"`
	Genesis   map[string]interface{} `yaml:"genesis"`
	Host      common.Host            `yaml:"host"`
	Version   string                 `yaml:"version"`
}

// AccountByName finds account by name.
func (c ConfigYaml) AccountByName(name string) (acc common.Account, found bool) {
	for _, acc := range c.Accounts {
		if acc.Name == name {
			return acc, true
		}
	}
	return common.Account{}, false
}

// GetVersion returns the version of the config.yaml file.
func (c ConfigYaml) GetVersion() string {
	return c.Version
}

// GetFaucet returns the Faucet.
func (c ConfigYaml) GetFaucet() common.Faucet {
	return c.Faucet
}

// ListAccounts returns the list of all the accounts.
func (c ConfigYaml) ListAccounts() []common.Account {
	return c.Accounts
}

// ListValidators returns the list of all the validators.
func (c ConfigYaml) ListValidators() []common.Validator {
	return []common.Validator{c.Validator}
}
