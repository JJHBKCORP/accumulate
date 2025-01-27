package cmd

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	net2 "net"
	"net/url"
	"os/exec"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	acctesting "gitlab.com/accumulatenetwork/accumulate/internal/testing"
)

type testCase func(t *testing.T, tc *testCmd)
type testMatrixTests []testCase

var testMatrix testMatrixTests

func bootstrap(t *testing.T, tc *testCmd) {
	//add the DN private key to our key list.
	_, err := tc.execute(t, fmt.Sprintf("key import private %x dnkey", tc.privKey.Bytes()))
	require.NoError(t, err)

	//set mnemonic for predictable addresses
	_, err = tc.execute(t, "key import mnemonic yellow yellow yellow yellow yellow yellow yellow yellow yellow yellow yellow yellow")
	require.NoError(t, err)

	//set the oracle price to $1.00
	_, err = tc.executeTx(t, "data write dn/oracle dnkey {\"price\":10000}")
	require.NoError(t, err)
}

func TestCli(t *testing.T) {
	acctesting.SkipLong(t)
	acctesting.SkipPlatformCI(t, "darwin", "flaky")

	tc := &testCmd{}
	tc.initalize(t)

	bootstrap(t, tc)

	testMatrix.execute(t, tc)

}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (tm *testMatrixTests) addTest(tc testCase) {
	*tm = append(*tm, tc)

}

func (tm *testMatrixTests) execute(t *testing.T, tc *testCmd) {

	// Sort by testCase number
	sort.SliceStable(*tm, func(i, j int) bool {
		return GetFunctionName((*tm)[i]) < GetFunctionName((*tm)[j])
	})

	//execute the tests
	for _, f := range testMatrix {
		name := strings.Split(GetFunctionName(f), ".")
		t.Run(name[len(name)-1], func(t *testing.T) { f(t, tc) })
	}
}

type testCmd struct {
	rootCmd        *cobra.Command
	directoryCmd   *exec.Cmd
	validatorCmd   *exec.Cmd
	defaultWorkDir string
	jsonRpcAddr    string
	privKey        crypto.PrivKey
}

//NewTestBVNN creates a BVN test Node and returns the rest and jsonrpc ports and the DN private key
func NewTestBVNN(t *testing.T) (string, crypto.PrivKey) {
	t.Helper()
	acctesting.SkipPlatformCI(t, "darwin", "requires setting up localhost aliases")

	// Start
	subnets, daemons := acctesting.CreateTestNet(t, 1, 1, 0)
	acctesting.RunTestNet(t, subnets, daemons)

	time.Sleep(time.Second)
	c := daemons[subnets[1]][0].Config

	return c.Accumulate.API.ListenAddress, daemons[subnets[0]][0].Key()
}

func (c *testCmd) initalize(t *testing.T) {
	t.Helper()

	c.rootCmd = InitRootCmd(initDB(t.TempDir(), true))
	c.rootCmd.PersistentPostRun = nil

	c.jsonRpcAddr, c.privKey = NewTestBVNN(t)
	time.Sleep(2 * time.Second)
}

func (c *testCmd) execute(t *testing.T, cmdLine string) (string, error) {
	fullCommand := fmt.Sprintf("-j -s %s/v2 %s",
		c.jsonRpcAddr, cmdLine)
	args := strings.Split(fullCommand, " ")

	e := bytes.NewBufferString("")
	b := bytes.NewBufferString("")
	c.rootCmd.SetErr(e)
	c.rootCmd.SetOut(b)
	c.rootCmd.SetArgs(args)
	DidError = nil
	c.rootCmd.Execute()
	if DidError != nil {
		return "", DidError
	}

	errPrint, err := ioutil.ReadAll(e)
	if err != nil {
		return "", err
	} else if len(errPrint) != 0 {
		return "", fmt.Errorf("%s", string(errPrint))
	}
	ret, err := ioutil.ReadAll(b)
	return string(ret), err
}

func (c *testCmd) executeTx(t *testing.T, cmdLine string, args ...interface{}) (string, error) {
	cmdLine = fmt.Sprintf(cmdLine, args...)
	out, err := c.execute(t, cmdLine)
	if err == nil {
		waitForTxns(t, c, out)
	}
	return out, err
}

// listenHttpUrl
// takes a string such as `http://localhost:123` and creates a TCP listener.
func listenHttpUrl(s string) (net2.Listener, bool, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, false, fmt.Errorf("invalid address: %v", err)
	}

	if u.Path != "" && u.Path != "/" {
		return nil, false, fmt.Errorf("invalid address: path is not empty")
	}

	var secure bool
	switch u.Scheme {
	case "tcp", "http":
		secure = false
	case "https":
		secure = true
	default:
		return nil, false, fmt.Errorf("invalid address: unsupported scheme %q", u.Scheme)
	}

	l, err := net2.Listen("tcp", u.Host)
	if err != nil {
		return nil, false, err
	}

	return l, secure, nil
}

func waitForTxns(t *testing.T, tc *testCmd, jsonRes string) {
	t.Helper()

	var res ActionResponse
	require.NoError(t, json.Unmarshal([]byte(jsonRes), &res))

	commandLine := fmt.Sprintf("tx get --wait 10s --wait-synth 10s %s", hex.EncodeToString(res.TransactionHash))
	_, err := tc.execute(t, commandLine)
	require.NoError(t, err)
}
