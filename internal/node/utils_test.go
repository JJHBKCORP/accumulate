package node_test

import (
	"fmt"
	"io"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/AccumulateNetwork/accumulate/config"
	cfg "github.com/AccumulateNetwork/accumulate/config"
	"github.com/AccumulateNetwork/accumulate/internal/abci"
	"github.com/AccumulateNetwork/accumulate/internal/api"
	"github.com/AccumulateNetwork/accumulate/internal/logging"
	"github.com/AccumulateNetwork/accumulate/internal/node"
	"github.com/AccumulateNetwork/accumulate/internal/relay"
	acctesting "github.com/AccumulateNetwork/accumulate/internal/testing"
	"github.com/AccumulateNetwork/accumulate/networks"
	"github.com/AccumulateNetwork/accumulate/types/state"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	rpc "github.com/tendermint/tendermint/rpc/client/http"
)

func initNodes(t *testing.T, name string, baseIP net.IP, basePort int, count int, relay []string) ([]*node.Node, []*state.StateDB) {
	t.Helper()

	IPs := make([]string, count)
	config := make([]*config.Config, count)

	for i := range IPs {
		ip := make(net.IP, len(baseIP))
		copy(ip, baseIP)
		ip[15] += byte(i)
		IPs[i] = fmt.Sprintf("tcp://%v", ip)
	}

	for i := range config {
		config[i] = cfg.Default(cfg.BlockValidator, cfg.Validator)
		if relay != nil {
			config[i].Accumulate.Networks = make([]string, len(relay))
			for j, r := range relay {
				config[i].Accumulate.Networks[j] = fmt.Sprintf("tcp://%s:%d", r, basePort+networks.TmRpcPortOffset)
			}
		} else {
			config[i].Accumulate.Networks = []string{fmt.Sprintf("%s:%d", IPs[0], basePort+networks.TmRpcPortOffset)}
		}
		config[i].Consensus.CreateEmptyBlocks = false
		config[i].Accumulate.API.EnableSubscribeTX = true
	}

	workDir := t.TempDir()
	require.NoError(t, node.Init(node.InitOptions{
		WorkDir:   workDir,
		ShardName: name,
		SubnetID:  name,
		Port:      basePort,
		Config:    config,
		RemoteIP:  IPs,
		ListenIP:  IPs,
	}))

	nodes := make([]*node.Node, count)
	dbs := make([]*state.StateDB, count)
	for i := range nodes {
		nodeDir := filepath.Join(workDir, fmt.Sprintf("Node%d", i))
		c, err := cfg.Load(nodeDir)
		require.NoError(t, err)

		c.Instrumentation.Prometheus = false
		c.Accumulate.WebsiteEnabled = false

		// Make a block every 1/10th second, to make tests go faster
		c.Consensus.TimeoutCommit = time.Second / 10

		require.NoError(t, cfg.Store(c))

		opts := acctesting.BVNNOptions{
			Dir:       nodeDir,
			LogWriter: logging.TestLogWriter(t),
			Logger: func(w io.Writer) zerolog.Logger {
				zl := zerolog.New(w)
				zl = zl.With().Int("node", i).Logger()
				zl = zl.Hook(logging.ExcludeMessages("starting service", "stopping service"))
				// zl = zl.Hook(logging.BodyHook(func(e *zerolog.Event, _ zerolog.Level, body map[string]interface{}) {
				// 	module, ok := body["module"].(string)
				// 	if !ok {
				// 		return
				// 	}

				// 	switch module {
				// 	case "rpc-server", "p2p", "rpc", "statesync":
				// 		e.Discard()
				// 	default:
				// 		e.Discard()
				// 	case "accumulate":
				// 		// OK
				// 	}
				// }))
				return zl
			},
		}

		nodes[i], dbs[i], _, err = acctesting.NewBVNN(opts, t.Cleanup)
		require.NoError(t, err)

		nodes[i].ABCI.(*abci.Accumulator).OnFatal(func(err error) { require.NoError(t, err) })
	}

	return nodes, dbs
}

func startNodes(t *testing.T, nodes []*node.Node) *api.Query {
	t.Helper()

	for _, n := range nodes {
		require.NoError(t, n.Start())
	}

	t.Cleanup(func() {
		wg := new(sync.WaitGroup)
		wg.Add(len(nodes))
		for _, n := range nodes {
			n := n // Do not capture the loop var
			go func() {
				defer wg.Done()
				_ = n.Stop()
				n.Wait()
			}()
		}
		wg.Wait()
	})

	rpc, err := rpc.New(nodes[0].Config.RPC.ListenAddress)
	require.NoError(t, err)

	relay := relay.New(rpc)
	if nodes[0].Config.Accumulate.API.EnableSubscribeTX {
		require.NoError(t, relay.Start())
		t.Cleanup(func() { require.NoError(t, relay.Stop()) })
	}

	return api.NewQuery(relay)
}
