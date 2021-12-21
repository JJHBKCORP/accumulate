// Code generated by MockGen. DO NOT EDIT.
// Source: types.go

// Package mock_api is a generated GoMock package.
package mock_api

import (
	context "context"
	"github.com/AccumulateNetwork/accumulate/internal/url"
	"github.com/AccumulateNetwork/accumulate/networks/connections"
	"github.com/AccumulateNetwork/accumulate/protocol"
	"github.com/tendermint/tendermint/rpc/client/http"
	"github.com/ybbus/jsonrpc/v2"
	reflect "reflect"
	time "time"

	api "github.com/AccumulateNetwork/accumulate/internal/api/v2"
	gomock "github.com/golang/mock/gomock"
	bytes "github.com/tendermint/tendermint/libs/bytes"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	types "github.com/tendermint/tendermint/types"
)

// MockQuerier is a mock of Querier interface.
type MockQuerier struct {
	ctrl     *gomock.Controller
	recorder *MockQuerierMockRecorder
}

// MockQuerierMockRecorder is the mock recorder for MockQuerier.
type MockQuerierMockRecorder struct {
	mock *MockQuerier
}

// NewMockQuerier creates a new mock instance.
func NewMockQuerier(ctrl *gomock.Controller) *MockQuerier {
	mock := &MockQuerier{ctrl: ctrl}
	mock.recorder = &MockQuerierMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockQuerier) EXPECT() *MockQuerierMockRecorder {
	return m.recorder
}

// QueryChain mocks base method.
func (m *MockQuerier) QueryChain(id []byte) (*api.QueryResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryChain", id)
	ret0, _ := ret[0].(*api.QueryResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryChain indicates an expected call of QueryChain.
func (mr *MockQuerierMockRecorder) QueryChain(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryChain", reflect.TypeOf((*MockQuerier)(nil).QueryChain), id)
}

// QueryData mocks base method.
func (m *MockQuerier) QueryData(url string, entryHash [32]byte) (*api.QueryResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryData", url, entryHash)
	ret0, _ := ret[0].(*api.QueryResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryData indicates an expected call of QueryData.
func (mr *MockQuerierMockRecorder) QueryData(url, entryHash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryData", reflect.TypeOf((*MockQuerier)(nil).QueryData), url, entryHash)
}

// QueryDataSet mocks base method.
func (m *MockQuerier) QueryDataSet(url string, pagination api.QueryPagination, opts api.QueryOptions) (*api.QueryResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryDataSet", url, pagination, opts)
	ret0, _ := ret[0].(*api.QueryResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryDataSet indicates an expected call of QueryDataSet.
func (mr *MockQuerierMockRecorder) QueryDataSet(url, pagination, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryDataSet", reflect.TypeOf((*MockQuerier)(nil).QueryDataSet), url, pagination, opts)
}

// QueryDirectory mocks base method.
func (m *MockQuerier) QueryDirectory(url string, pagination api.QueryPagination, opts api.QueryOptions) (*api.QueryResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryDirectory", url, pagination, opts)
	ret0, _ := ret[0].(*api.QueryResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryDirectory indicates an expected call of QueryDirectory.
func (mr *MockQuerierMockRecorder) QueryDirectory(url, pagination, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryDirectory", reflect.TypeOf((*MockQuerier)(nil).QueryDirectory), url, pagination, opts)
}

// QueryKeyPageIndex mocks base method.
func (m *MockQuerier) QueryKeyPageIndex(url string, key []byte) (*api.QueryResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryKeyPageIndex", url, key)
	ret0, _ := ret[0].(*api.QueryResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryKeyPageIndex indicates an expected call of QueryKeyPageIndex.
func (mr *MockQuerierMockRecorder) QueryKeyPageIndex(url, key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryKeyPageIndex", reflect.TypeOf((*MockQuerier)(nil).QueryKeyPageIndex), url, key)
}

// QueryTx mocks base method.
func (m *MockQuerier) QueryTx(id []byte, wait time.Duration) (*api.QueryResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryTx", id, wait)
	ret0, _ := ret[0].(*api.QueryResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryTx indicates an expected call of QueryTx.
func (mr *MockQuerierMockRecorder) QueryTx(id, wait interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryTx", reflect.TypeOf((*MockQuerier)(nil).QueryTx), id, wait)
}

// QueryTxHistory mocks base method.
func (m *MockQuerier) QueryTxHistory(url string, start, count uint64) (*api.QueryMultiResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryTxHistory", url, start, count)
	ret0, _ := ret[0].(*api.QueryMultiResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryTxHistory indicates an expected call of QueryTxHistory.
func (mr *MockQuerierMockRecorder) QueryTxHistory(url, start, count interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryTxHistory", reflect.TypeOf((*MockQuerier)(nil).QueryTxHistory), url, start, count)
}

// QueryUrl mocks base method.
func (m *MockQuerier) QueryUrl(url string) (*api.QueryResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryUrl", url)
	ret0, _ := ret[0].(*api.QueryResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryUrl indicates an expected call of QueryUrl.
func (mr *MockQuerierMockRecorder) QueryUrl(url interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryUrl", reflect.TypeOf((*MockQuerier)(nil).QueryUrl), url)
}

// MockABCIQueryClient is a mock of ABCIQueryClient interface.
type MockABCIQueryClient struct {
	ctrl     *gomock.Controller
	recorder *MockABCIQueryClientMockRecorder
}

// MockABCIQueryClientMockRecorder is the mock recorder for MockABCIQueryClient.
type MockABCIQueryClientMockRecorder struct {
	mock *MockABCIQueryClient
}

// NewMockABCIQueryClient creates a new mock instance.
func NewMockABCIQueryClient(ctrl *gomock.Controller) *MockABCIQueryClient {
	mock := &MockABCIQueryClient{ctrl: ctrl}
	mock.recorder = &MockABCIQueryClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockABCIQueryClient) EXPECT() *MockABCIQueryClientMockRecorder {
	return m.recorder
}

// ABCIQuery mocks base method.
func (m *MockABCIQueryClient) ABCIQuery(ctx context.Context, path string, data bytes.HexBytes) (*coretypes.ResultABCIQuery, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ABCIQuery", ctx, path, data)
	ret0, _ := ret[0].(*coretypes.ResultABCIQuery)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ABCIQuery indicates an expected call of ABCIQuery.
func (mr *MockABCIQueryClientMockRecorder) ABCIQuery(ctx, path, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ABCIQuery", reflect.TypeOf((*MockABCIQueryClient)(nil).ABCIQuery), ctx, path, data)
}

// MockABCIBroadcastClient is a mock of ABCIBroadcastClient interface.
type MockABCIBroadcastClient struct {
	ctrl     *gomock.Controller
	recorder *MockABCIBroadcastClientMockRecorder
}

func (m *MockABCIBroadcastClient) NewBatch() *http.BatchHTTP {
	return nil
}

// MockConnectionRouter is a mock of ConnectionRouter interface.
type MockConnectionRouter struct {
	route    MockRoute
	recorder *MockConnectionRouterMockRecorder
}

// MockConnectionRouter is a mock of ConnectionRouter interface.
type MockRoute struct {
	broadcastClient *MockABCIBroadcastClient
	subnetName      string
	networkGroup    connections.NetworkGroup
	client          jsonrpc.RPCClient
}

func (m MockRoute) GetSubnetName() string {
	return m.subnetName
}

func (m MockRoute) GetJsonRpcClient() jsonrpc.RPCClient {
	return m.client
}

func (m MockRoute) GetQueryClient() connections.ABCIQueryClient {
	return nil
}

func (m MockRoute) GetBroadcastClient() connections.ABCIBroadcastClient {
	return m.broadcastClient
}

func (m MockRoute) IsDirectoryNode() bool {
	return false
}

func (m MockRoute) GetNetworkGroup() connections.NetworkGroup {
	return m.networkGroup
}

func (m MockRoute) GetBatchBroadcastClient() connections.BatchABCIBroadcastClient {
	return m.broadcastClient
}

func (m *MockConnectionRouter) SelectRoute(adiUrl *url.URL, allowFollower bool) (connections.Route, error) {
	if protocol.IsDnUrl(adiUrl) {
		return nil, nil
	}
	return m.route, nil
}

func (m *MockConnectionRouter) GetLocalRoute() (connections.Route, error) {
	return m.route, nil
}

func (m *MockConnectionRouter) GetAll() ([]connections.Route, error) {
	return []connections.Route{m.route}, nil
}

func (m *MockConnectionRouter) GetAllBVNs() ([]connections.Route, error) {
	return []connections.Route{m.route}, nil
}

// MockABCIBroadcastClientMockRecorder is the mock recorder for MockABCIBroadcastClient.
type MockABCIBroadcastClientMockRecorder struct {
	mock *MockABCIBroadcastClient
}

// NewMockABCIBroadcastClient creates a new mock instance.
func NewMockABCIBroadcastClient(ctrl *gomock.Controller) *MockABCIBroadcastClient {
	mock := &MockABCIBroadcastClient{ctrl: ctrl}
	mock.recorder = &MockABCIBroadcastClientMockRecorder{mock}
	return mock
}

// MockAonnectionRouterMockRecorder is the mock recorder for MockConnectionRouter.
type MockConnectionRouterMockRecorder struct {
	mock *MockConnectionRouter
}

func NewMockConnectionRouter(broadcastClient *MockABCIBroadcastClient) *MockConnectionRouter {
	mock := &MockConnectionRouter{route: MockRoute{
		broadcastClient: broadcastClient,
		subnetName:      "testnet",
		networkGroup:    connections.OtherSubnet,
		client:          jsonrpc.NewClient("http://localhost"),
	}}
	mock.recorder = &MockConnectionRouterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockABCIBroadcastClient) EXPECT() *MockABCIBroadcastClientMockRecorder {
	return m.recorder
}

// BroadcastTxAsync mocks base method.
func (m *MockABCIBroadcastClient) BroadcastTxAsync(arg0 context.Context, arg1 types.Tx) (*coretypes.ResultBroadcastTx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BroadcastTxAsync", arg0, arg1)
	ret0, _ := ret[0].(*coretypes.ResultBroadcastTx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BroadcastTxAsync indicates an expected call of BroadcastTxAsync.
func (mr *MockABCIBroadcastClientMockRecorder) BroadcastTxAsync(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BroadcastTxAsync", reflect.TypeOf((*MockABCIBroadcastClient)(nil).BroadcastTxAsync), arg0, arg1)
}

// BroadcastTxSync mocks base method.
func (m *MockABCIBroadcastClient) BroadcastTxSync(arg0 context.Context, arg1 types.Tx) (*coretypes.ResultBroadcastTx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BroadcastTxSync", arg0, arg1)
	ret0, _ := ret[0].(*coretypes.ResultBroadcastTx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BroadcastTxSync indicates an expected call of BroadcastTxSync.
func (mr *MockABCIBroadcastClientMockRecorder) BroadcastTxSync(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BroadcastTxSync", reflect.TypeOf((*MockABCIBroadcastClient)(nil).BroadcastTxSync), arg0, arg1)
}

// CheckTx mocks base method.
func (m *MockABCIBroadcastClient) CheckTx(ctx context.Context, tx types.Tx) (*coretypes.ResultCheckTx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckTx", ctx, tx)
	ret0, _ := ret[0].(*coretypes.ResultCheckTx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckTx indicates an expected call of CheckTx.
func (mr *MockABCIBroadcastClientMockRecorder) CheckTx(ctx, tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckTx", reflect.TypeOf((*MockABCIBroadcastClient)(nil).CheckTx), ctx, tx)
}
