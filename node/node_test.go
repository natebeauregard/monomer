package node_test

import (
	"context"
	cometdb "github.com/cometbft/cometbft-db"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/polymerdao/monomer"
	"github.com/polymerdao/monomer/environment"
	"github.com/polymerdao/monomer/genesis"
	"github.com/polymerdao/monomer/node"
	"github.com/polymerdao/monomer/testapp"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"net/http"
	"testing"
)

func TestRun(t *testing.T) {
	chainID := monomer.ChainID(0)
	engineWS, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	cometListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	app := testapp.NewTest(t, chainID.String())
	blockdb := dbm.NewMemDB()
	defer func() {
		require.NoError(t, blockdb.Close())
	}()
	txdb := cometdb.NewMemDB()
	defer func() {
		require.NoError(t, txdb.Close())
	}()
	mempooldb := dbm.NewMemDB()
	defer func() {
		require.NoError(t, mempooldb.Close())
	}()
	metrics, err := telemetry.New(telemetry.Config{Enabled: true})
	if err != nil {
		require.NoError(t, err)
	}
	n := node.New(
		app,
		&genesis.Genesis{
			ChainID:  chainID,
			AppState: testapp.MakeGenesisAppState(t, app),
		},
		engineWS,
		cometListener,
		blockdb,
		mempooldb,
		txdb,
		&node.SelectiveListener{
			OnEngineHTTPServeErrCb: func(err error) {
				require.NoError(t, err)
			},
			OnEngineWebsocketServeErrCb: func(err error) {
				require.NoError(t, err)
			},
		},
		metrics,
	)

	env := environment.New()
	defer func() {
		require.NoError(t, env.Close())
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, n.Run(ctx, env))

	client, err := rpc.DialContext(ctx, "ws://"+engineWS.Addr().String())
	require.NoError(t, err)
	defer client.Close()
	ethClient := ethclient.NewClient(client)
	chainIDBig, err := ethClient.ChainID(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(chainID), chainIDBig.Uint64())

	cometClient, err := rpc.DialContext(ctx, "http://"+cometListener.Addr().String())
	require.NoError(t, err)
	defer cometClient.Close()
	want := "hello, world"
	var msg string
	require.NoError(t, cometClient.Call(&msg, "echo", want))
	require.Equal(t, want, msg)

	// TODO: remove from this test or add to a separate test - testing purposes only
	//resp, err := http.Get("http://127.0.0.1:26660/metrics")
	resp, err := http.Get("http://127.0.0.1:8892/metrics")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	respBodyBz, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()
	respBody := string(respBodyBz)
	t.Log("Monomer metrics response: ", respBody)
	require.Contains(t, respBody, "eth.chain_id")
}
