package config

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pelletier/go-toml"
	"github.com/spf13/viper"
	tm "github.com/tendermint/tendermint/config"
	accurl "gitlab.com/accumulatenetwork/accumulate/internal/url"
	"gitlab.com/accumulatenetwork/accumulate/protocol"
)

type NetworkType string

const (
	BlockValidator NetworkType = "block-validator"
	Directory      NetworkType = "directory"
)

type NodeType string

const (
	Validator NodeType = "validator"
	Follower  NodeType = "follower"
)

const DefaultLogLevels = "error;accumulate=info" // main=info;state=info;statesync=info;accumulate=debug;executor=info;disk-monitor=info;init=info

func Default(net NetworkType, node NodeType, netId string) *Config {
	c := new(Config)
	c.Accumulate.Network.Type = net
	c.Accumulate.Network.LocalSubnetID = netId
	c.Accumulate.API.PrometheusServer = "http://18.119.26.7:9090"
	c.Accumulate.SentryDSN = "https://glet_78c3bf45d009794a4d9b0c990a1f1ed5@gitlab.com/api/v4/error_tracking/collector/29762666"
	c.Accumulate.Website.Enabled = true
	c.Accumulate.API.TxMaxWaitTime = 10 * time.Second
	c.Accumulate.API.EnableDebugMethods = true
	switch node {
	case Validator:
		c.Config = *tm.DefaultValidatorConfig()
	default:
		c.Config = *tm.DefaultConfig()
	}
	c.LogLevel = DefaultLogLevels
	c.Instrumentation.Prometheus = true
	c.ProxyApp = ""
	return c
}

type Config struct {
	tm.Config
	Accumulate Accumulate
}

type Accumulate struct {
	SentryDSN string `toml:"sentry-dsn" mapstructure:"sentry-dsn"`

	Network Network `toml:"network" mapstructure:"network"`
	API     API     `toml:"api" mapstructure:"api"`
	Website Website `toml:"website" mapstructure:"website"`
}

type Network struct {
	Type          NetworkType `toml:"type" mapstructure:"type"`
	LocalSubnetID string      `toml:"local-subnet" mapstructure:"local-subnet"`
	LocalAddress  string      `toml:"local-address" mapstructure:"local-address"`
	Subnets       []Subnet    `toml:"subnets" mapstructure:"subnets"`
}

type Subnet struct {
	ID    string      `toml:"id" mapstructure:"id"`
	Type  NetworkType `toml:"type" mapstructure:"type"`
	Nodes []Node      `toml:"nodes" mapstructure:"nodes"`
}

type Node struct {
	Address string
	Type    NodeType `toml:"type" mapstructure:"type"`
}

type API struct {
	TxMaxWaitTime      time.Duration `toml:"tx-max-wait-time" mapstructure:"tx-max-wait-time"`
	PrometheusServer   string        `toml:"prometheus-server" mapstructure:"prometheus-server"`
	ListenAddress      string        `toml:"listen-address" mapstructure:"listen-address"`
	DebugJSONRPC       bool          `toml:"debug-jsonrpc" mapstructure:"debug-jsonrpc"`
	EnableDebugMethods bool          `toml:"enable-debug-methods" mapstructure:"enable-debug-methods"`
}

type Website struct {
	Enabled       bool   `toml:"website-enabled" mapstructure:"website-enabled"`
	ListenAddress string `toml:"website-listen-address" mapstructure:"website-listen-address"`
}

func OffsetPort(addr string, offset int) (*url.URL, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %v", addr, err)
	}
	if u.Scheme == "" {
		return nil, fmt.Errorf("invalid URL %q: has no scheme, so this probably isn't a URL", addr)
	}

	port, err := strconv.ParseInt(u.Port(), 10, 17)
	if err != nil {
		return nil, fmt.Errorf("invalid port %q: %v", u.Port(), err)
	}

	port += int64(offset)
	u.Host = fmt.Sprintf("%s:%d", u.Hostname(), port)
	return u, nil
}

func (n *Network) NodeUrl(path ...string) *accurl.URL {
	if n.Type == Directory {
		return protocol.DnUrl().JoinPath(path...)
	}

	return protocol.BvnUrl(n.LocalSubnetID).JoinPath(path...)
}

func (n *Network) GetBvnNames() []string {
	var names []string
	for _, subnet := range n.Subnets {
		if subnet.Type == BlockValidator {
			names = append(names, subnet.ID)
		}
	}
	return names
}

func (n *Network) GetSubnetByID(subnetID string) Subnet {
	for _, subnet := range n.Subnets {
		if subnet.ID == subnetID {
			return subnet
		}
	}
	panic(fmt.Sprintf("Subnet ID %s does not exist", subnetID))
}

func Load(dir string) (*Config, error) {
	return loadFile(dir, filepath.Join(dir, "config", "config.toml"), filepath.Join(dir, "config", "accumulate.toml"))
}

func loadFile(dir, tmFile, accFile string) (*Config, error) {
	tm, err := loadTendermint(dir, tmFile)
	if err != nil {
		return nil, err
	}

	acc, err := loadAccumulate(dir, accFile)
	if err != nil {
		return nil, err
	}

	return &Config{*tm, *acc}, nil
}

func Store(config *Config) error {
	// Exits on fail, hard-coded to write to '${config.RootDir}/config/config.toml'
	tm.WriteConfigFile(config.RootDir, &config.Config)

	f, err := os.Create(filepath.Join(config.RootDir, "config", "accumulate.toml"))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	return toml.NewEncoder(f).Encode(config.Accumulate)
}

func loadTendermint(dir, file string) (*tm.Config, error) {
	config := tm.DefaultConfig()
	err := load(dir, file, config)
	if err != nil {
		return nil, err
	}

	config.SetRoot(dir)
	tm.EnsureRoot(config.RootDir)
	if err := config.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("validate: %v", err)
	}
	return config, nil
}

func loadAccumulate(dir, file string) (*Accumulate, error) {
	config := new(Accumulate)
	err := load(dir, file, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func load(dir, file string, c interface{}) error {
	v := viper.New()
	v.SetConfigFile(file)
	v.AddConfigPath(dir)
	err := v.ReadInConfig()
	if err != nil {
		return fmt.Errorf("read: %v", err)
	}

	err = v.Unmarshal(c)
	if err != nil {
		return fmt.Errorf("unmarshal: %v", err)
	}

	return nil
}
