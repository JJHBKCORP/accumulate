package api

import (
	"context"
	"encoding/json"
	"github.com/AccumulateNetwork/accumulate/networks/connections"
	stdlog "log"
	"net/http"
	"os"
	"time"

	"github.com/AccumulateNetwork/accumulate"
	"github.com/AccumulateNetwork/accumulate/config"
	v1 "github.com/AccumulateNetwork/accumulate/internal/api"
	"github.com/AccumulateNetwork/accumulate/protocol"
	"github.com/AccumulateNetwork/jsonrpc2/v15"
	"github.com/go-playground/validator/v10"
	"github.com/tendermint/tendermint/libs/log"
)

type JrpcOptions struct {
	Config           *config.API
	Query            Querier
	ConnectionRouter connections.ConnectionRouter
	QueueDuration    time.Duration

	QueueDepth int
	Logger     log.Logger
	// Deprecated: will be removed when API v1 is removed
	QueryV1 *v1.Query
}

type JrpcMethods struct {
	methods  jsonrpc2.MethodMap
	opts     JrpcOptions
	validate *validator.Validate
	lclRoute connections.Route
	exch     chan executeRequest
	queue    executeQueue
	logger   log.Logger

	// Deprecated: will be removed
	v1 *v1.API
}

func NewJrpc(opts JrpcOptions) (*JrpcMethods, error) {
	var err error
	m := new(JrpcMethods)
	m.opts = opts
	m.exch = make(chan executeRequest)
	m.queue.leader = make(chan struct{}, 1)
	m.queue.leader <- struct{}{}
	m.queue.enqueue = make(chan *executeRequest)

	m.lclRoute, err = m.opts.ConnectionRouter.GetLocalRoute()
	if err != nil {
		return nil, err
	}

	if opts.Logger != nil {
		m.logger = opts.Logger.With("module", "jrpc")
	}

	m.validate, err = protocol.NewValidator()
	if err != nil {
		return nil, err
	}

	m.v1, err = v1.New(opts.Config, opts.QueryV1)
	if err != nil {
		return nil, err
	}

	if opts.Config != nil && opts.Config.DebugJSONRPC {
		jsonrpc2.DebugMethodFunc = true
	}

	m.populateMethodTable()
	return m, nil
}

func (m *JrpcMethods) logDebug(msg string, keyVals ...interface{}) {
	if m.logger != nil {
		m.logger.Debug(msg, keyVals...)
	}
}

func (m *JrpcMethods) logError(msg string, keyVals ...interface{}) {
	if m.logger != nil {
		m.logger.Error(msg, keyVals...)
	}
}

func (m *JrpcMethods) EnableDebug() error {

	q := &queryDirect{connRoute: m.lclRoute}

	m.methods["debug-query-direct"] = func(_ context.Context, params json.RawMessage) interface{} {
		req := new(GeneralQuery)
		err := m.parse(params, req)
		if err != nil {
			return err
		}

		return jrpcFormatResponse(q.QueryUrl(req.Url, req.QueryOptions))
	}
	return nil
}

func (m *JrpcMethods) NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/v1", m.v1.Handler())
	mux.Handle("/v2", jsonrpc2.HTTPRequestHandler(m.methods, stdlog.New(os.Stdout, "", 0)))
	return mux
}

func (m *JrpcMethods) Version(_ context.Context, params json.RawMessage) interface{} {
	res := new(ChainQueryResponse)
	res.Type = "version"
	res.Data = map[string]interface{}{
		"version":        accumulate.Version,
		"commit":         accumulate.Commit,
		"versionIsKnown": accumulate.IsVersionKnown(),
	}
	return res
}

func (m *JrpcMethods) Querier() Querier {
	return m.opts.Query
}
