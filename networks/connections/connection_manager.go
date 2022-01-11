package connections

import (
	"context"
	"fmt"
	"github.com/AccumulateNetwork/accumulate/config"
	"github.com/AccumulateNetwork/accumulate/networks"
	"github.com/AccumulateNetwork/accumulate/protocol"
	"github.com/AccumulateNetwork/accumulate/types/api/query"
	"github.com/tendermint/tendermint/libs/log"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	"github.com/tendermint/tendermint/rpc/client/local"
	"github.com/ybbus/jsonrpc/v2"
	"net"
	neturl "net/url"
	"strings"
	"time"
)

const UnhealthyNodeCheckInterval = time.Minute * 10 // TODO Configurable in toml?

type ConnectionManager interface {
	getBVNContextMap() map[string][]*nodeContext
	getDNContextList() []*nodeContext
	getFNContextList() []*nodeContext
	GetLocalNodeContext() *nodeContext
	GetLocalClient() *local.Local
}

type ConnectionInitializer interface {
	CreateClients(*local.Local) error
}

type connectionManager struct {
	accConfig    *config.Accumulate
	bvnCtxMap    map[string][]*nodeContext
	dnCtxList    []*nodeContext
	fnCtxList    []*nodeContext
	localNodeCtx *nodeContext
	localClient  *local.Local
	logger       log.Logger
}

func (cm *connectionManager) doHealthCheckOnNode(nc *nodeContext) {
	// Try to get the version using the jsonRpcClient
	/*	FIXME this call does not work.  Maybe only on v1?
		_, err := nc.jsonRpcClient.Call("version")
		if err != nil {
			nc.ReportError(err)
			return
		}
	*/

	// Try to query Tendermint with something it should not find
	qu := query.Query{}
	qd, _ := qu.MarshalBinary()
	qryRes, err := nc.GetQueryClient().ABCIQuery(context.Background(), "/abci_query", qd)
	if err != nil || qryRes.Response.Code != 19 { // FIXME code 19 will emit an error in the log
		nc.ReportError(err)
		if qryRes != nil {
			cm.logger.Info("ABCIQuery response: %v", qryRes.Response)
		}
		return
	}

	/*	FIXME this call does not work Maybe only on v1?
		res, err := nc.jsonRpcClient.Call("metrics", &protocol.MetricsRequest{Metric: "tps", Duration: time.Hour})
		cm.logger.Info("TPS response: %v", res.Result)
	*/
	nc.metrics.status = Up
}

type nodeMetrics struct {
	status NodeStatus
	// TODO add metrics that can be useful for the router to determine whether it should put or should avoid putting put more load on a BVN
}

func NewConnectionManager(accConfig *config.Accumulate, logger log.Logger) ConnectionManager {
	cm := new(connectionManager)
	cm.accConfig = accConfig
	cm.logger = logger
	cm.buildNodeInventory()
	return cm
}

func (cm *connectionManager) getBVNContextMap() map[string][]*nodeContext {
	return cm.bvnCtxMap
}

func (cm *connectionManager) getDNContextList() []*nodeContext {
	return cm.dnCtxList
}

func (cm *connectionManager) getFNContextList() []*nodeContext {
	return cm.fnCtxList
}

func (cm *connectionManager) GetLocalNodeContext() *nodeContext {
	return cm.localNodeCtx
}

func (cm *connectionManager) GetLocalClient() *local.Local {
	return cm.localClient
}

func (cm *connectionManager) buildNodeInventory() {
	cm.bvnCtxMap = make(map[string][]*nodeContext)

	for subnetName, addresses := range cm.accConfig.Network.Addresses {
		for _, address := range addresses {
			nodeCtx, err := cm.buildNodeContext(address, subnetName)
			if err != nil {
				cm.logger.Error("error building node context for node %s on net %s with type %s: %w, ignoring node...",
					nodeCtx.address, nodeCtx.subnetName, nodeCtx.nodeType, err)
				continue
			}

			switch nodeCtx.nodeType {
			case config.Validator:
				switch nodeCtx.netType {
				case config.BlockValidator:
					bvnName := protocol.BvnNameFromSubnetName(subnetName)
					nodeList, ok := cm.bvnCtxMap[bvnName]
					if !ok {
						nodeList := make([]*nodeContext, 1)
						nodeList[0] = nodeCtx
						cm.bvnCtxMap[bvnName] = nodeList
					} else {
						cm.bvnCtxMap[bvnName] = append(nodeList, nodeCtx)
					}
				case config.Directory:
					cm.dnCtxList = append(cm.dnCtxList, nodeCtx)
				}
			case config.Follower:
				cm.fnCtxList = append(cm.fnCtxList, nodeCtx)
			}
			if nodeCtx.networkGroup == Local {
				cm.localNodeCtx = nodeCtx
			}
		}
	}
}

func (cm *connectionManager) buildNodeContext(address string, subnetName string) (*nodeContext, error) {
	nodeCtx := &nodeContext{subnetName: subnetName,
		address: address,
		connMgr: cm,
		metrics: nodeMetrics{status: Unknown}}
	nodeCtx.networkGroup = cm.determineNetworkGroup(subnetName, address)
	nodeCtx.netType, nodeCtx.nodeType = determineTypes(subnetName, cm.accConfig.Network)

	if address != "local" && address != "self" {
		var err error
		nodeCtx.resolvedIPs, err = resolveIPs(address)
		if err != nil {
			nodeCtx.ReportErrorStatus(Down, fmt.Errorf("error resolving IPs for %s: %w", address, err))
		}
	}
	return nodeCtx, nil
}

func (cm *connectionManager) determineNetworkGroup(subnetName string, address string) NetworkGroup {
	switch {
	case (strings.EqualFold(subnetName, cm.accConfig.Network.ID) && strings.EqualFold(address, cm.accConfig.Network.SelfAddress)) ||
		strings.EqualFold(address, "local") || strings.EqualFold(address, "self"):
		return Local
	case strings.EqualFold(subnetName, cm.accConfig.Network.ID):
		return SameSubnet
	default:
		return OtherSubnet
	}
}

func determineTypes(subnetName string, netCfg config.Network) (config.NetworkType, config.NodeType) {
	var networkType config.NetworkType
	for _, bvnName := range netCfg.BvnNames {
		if strings.EqualFold(bvnName, subnetName) {
			networkType = config.BlockValidator
		}
	}
	if len(networkType) == 0 { // When it's not a block validator the only option is directory
		networkType = config.Directory
	}

	var nodeType config.NodeType
	nodeType = config.Validator // TODO follower support
	return networkType, nodeType
}

func (cm *connectionManager) CreateClients(lclClient *local.Local) error {
	cm.localClient = lclClient

	for _, nodeCtxList := range cm.bvnCtxMap {
		for _, nodeCtx := range nodeCtxList {
			err := cm.createJsonRpcClient(nodeCtx)
			if err != nil {
				return err
			}
			err = cm.createAbciClients(nodeCtx)
			if err != nil {
				return err
			}
		}
	}
	for _, nodeCtx := range cm.dnCtxList {
		err := cm.createJsonRpcClient(nodeCtx)
		if err != nil {
			return err
		}
		err = cm.createAbciClients(nodeCtx)
		if err != nil {
			return err
		}
	}
	for _, nodeCtx := range cm.fnCtxList {
		err := cm.createJsonRpcClient(nodeCtx)
		if err != nil {
			return err
		}
		err = cm.createAbciClients(nodeCtx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cm *connectionManager) createAbciClients(nodeCtx *nodeContext) error {
	switch nodeCtx.networkGroup {
	case Local:
		nodeCtx.queryClient = cm.localClient
		nodeCtx.broadcastClient = cm.localClient
	default:
		offsetAddr, err := config.OffsetPort(nodeCtx.address, networks.TmRpcPortOffset)
		if err != nil {
			return fmt.Errorf("invalid BVN address: %v", err)
		}
		client, err := rpchttp.New(offsetAddr)
		if err != nil {
			return fmt.Errorf("failed to create RPC client: %v", err)
		}

		nodeCtx.queryClient = client
		nodeCtx.broadcastClient = client
		nodeCtx.batchBroadcastClient = client
	}
	return nil
}

func (cm *connectionManager) createJsonRpcClient(nodeCtx *nodeContext) error {
	// RPC HTTP client
	var address string
	if strings.EqualFold(nodeCtx.address, "local") || strings.EqualFold(nodeCtx.address, "self") {
		address = cm.accConfig.API.ListenAddress
	} else {
		var err error
		address, err = config.OffsetPort(nodeCtx.address, networks.AccRouterJsonPortOffset)
		if err != nil {
			return fmt.Errorf("invalid BVN address: %v", err)
		}
	}

	nodeCtx.jsonRpcClient = jsonrpc.NewClient(address + "/v2")
	return nil
}

func resolveIPs(address string) ([]net.IP, error) {
	var hostname string
	nodeUrl, err := neturl.Parse(address)
	if err == nil {
		hostname = nodeUrl.Hostname()
	} else {
		hostname = address
	}

	ip := net.ParseIP(hostname)
	if ip != nil {
		return []net.IP{ip}, nil
	}

	/* TODO
	consider using DNS resolver with DNSSEC support like go-resolver and query directly to a DNS server list that supports this, like 1.1.1.1
	*/
	ipList, err := net.LookupIP(hostname)
	if err != nil {
		return nil, fmt.Errorf("error doing DNS lookup for %s: %w", address, err)
	}
	return ipList, nil
}
