package node

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	"net/http"
	"net/url"
	"time"

	"github.com/AccumulateNetwork/accumulated/config"
	web "github.com/AccumulateNetwork/accumulated/internal/web/static"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/rpc/client/local"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	coregrpc "github.com/tendermint/tendermint/rpc/grpc"
	rpcclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"
)

// AppFactory creates and returns an ABCI application.
type AppFactory func(*privval.FilePV) (abci.Application, error)

// Node wraps a Tendermint node.
type Node struct {
	service.Service
	Config      *config.Config
	APIClient   coregrpc.BroadcastAPIClient
	LocalClient *local.Local
}

// New initializes a Tendermint node for the given ABCI application.
func New(config *config.Config, app abci.Application) (*Node, error) {
	node := new(Node)
	node.Config = config

	// create logger
	var err error
	logger, err := log.NewDefaultLogger(config.LogFormat, config.LogLevel, false)
	if err != nil {
		return nil, fmt.Errorf("failed to parse log level: %w", err)
	}

	// create node
	node.Service, err = nm.New(&config.Config, logger, proxy.NewLocalClientCreator(app), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new Tendermint node: %w", err)
	}

	return node, nil
}

// Start starts the Tendermint node.
func (n *Node) Start() error {
	err := n.Service.Start()
	if err != nil {
		return err
	}

	if n.Config.Accumulate.WebsiteEnabled {
		u, err := url.Parse(n.Config.Accumulate.WebsiteListenAddress)
		if err != nil {
			return fmt.Errorf("invalid website listen address: %v", err)
		}
		if u.Scheme != "tcp" {
			return fmt.Errorf("invalid website listen address: expected scheme tcp, got %q", u.Scheme)
		}

		website := http.Server{Addr: u.Host, Handler: http.FileServer(http.FS(web.FS))}
		go func() {
			<-n.Quit()
			website.Shutdown(context.Background())
		}()
		go func() {
			stdlog.Printf("Starting website on %s", u.Host)
			err := website.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				stdlog.Fatalf("Failed to start website: %v", err)
			}
		}()
	}

	localns, ok := n.Service.(local.NodeService)
	if !ok {
		return fmt.Errorf("node cannot be used as a local node service")
	}

	n.LocalClient, err = local.New(localns)
	if err != nil {
		return fmt.Errorf("failed to create local client: %w", err)
	}

	n.APIClient = n.waitForGRPC()
	return n.waitForRPC()
}

func (n *Node) waitForGRPC() coregrpc.BroadcastAPIClient {
	client := coregrpc.StartGRPCClient(n.Config.RPC.GRPCListenAddress)
	for {
		_, err := client.Ping(context.Background(), &coregrpc.RequestPing{})
		if err == nil {
			return client
		}
	}
}

func (n *Node) waitForRPC() error {
	client, err := rpcclient.New(n.Config.RPC.ListenAddress)
	if err != nil {
		return err
	}

	result := new(ctypes.ResultStatus)
	for {
		_, err := client.Call(context.Background(), "status", map[string]interface{}{}, result)
		if err == nil {
			return nil
		}

		time.Sleep(time.Millisecond)
	}
}