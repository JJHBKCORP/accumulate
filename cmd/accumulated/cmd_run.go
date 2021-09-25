package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AccumulateNetwork/accumulated/blockchain/validator"
	"github.com/AccumulateNetwork/accumulated/config"
	"github.com/AccumulateNetwork/accumulated/internal/abci"
	"github.com/AccumulateNetwork/accumulated/internal/node"
	"github.com/AccumulateNetwork/accumulated/internal/relay"
	"github.com/AccumulateNetwork/accumulated/router"
	"github.com/spf13/cobra"
	tmabci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/privval"
	tmdb "github.com/tendermint/tm-db"
)

var cmdRun = &cobra.Command{
	Use:   "run",
	Short: "Run node",
	Run:   runNode,
}

var flagRun struct {
	Node int
}

func init() {
	cmdMain.AddCommand(cmdRun)

	cmdRun.Flags().IntVarP(&flagRun.Node, "node", "n", -1, "Which node are we? [0, n)")
}

func runNode(cmd *cobra.Command, args []string) {
	workDir := flagMain.WorkDir
	if cmd.Flag("node").Changed {
		nodeDir := fmt.Sprintf("Node%d", flagRun.Node)
		workDir = filepath.Join(workDir, nodeDir)
	} else if !cmd.Flag("work-dir").Changed {
		fmt.Fprint(os.Stderr, "Error: at least one of --work-dir or --node is required\n")
		_ = cmd.Usage()
		os.Exit(1)
	}

	config, err := config.Load(workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: reading config file: %v\n", err)
		os.Exit(1)
	}

	// Load state DB
	db, err := tmdb.NewGoLevelDB("kvstore", config.RootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create GoLevelDB: %v", err)
		os.Exit(1)
	}

	// Create node
	node, err := node.New(config, func(pv *privval.FilePV) (tmabci.Application, error) {
		vnode := new(validator.Node)
		vchain := &validator.NewBlockValidatorChain().ValidatorContext
		err = vnode.Initialize(config, pv.Key.PrivKey.Bytes(), vchain)
		if err != nil {
			return nil, err
		}

		return abci.NewAccumulator(db, pv, vnode)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize node: %v\n", err)
		os.Exit(1)
	}

	// Start node
	err = node.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to start node: %v\n", err)
		os.Exit(1)
	}

	///we really need to open up ports to ALL shards in the system.  Maybe this should be a query to the DBVC blockchain.
	txRelay := relay.NewWithNetworks(3)

	//the query object connects to the BVC, will be replaced with network client router
	query := router.NewQuery(txRelay)

	router.StartAPI(&config.Accumulate.AccRouter, query, txRelay)

	//Block forever
	select {}
}