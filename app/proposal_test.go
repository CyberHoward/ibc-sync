package app_test

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/go-bip39"
	"github.com/fatal-fruit/cosmapp/app"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

const (
	keyringPassphrase = "testpassphrase"
	keyringAppName    = "cosmappd"
)

type testFixture struct {
	ctx sdk.Context
	k   authkeeper.AccountKeeper

	addrs []sdk.AccAddress
}

//func initFixture(t *testing.T) *testFixture {
//	encCfg := moduletestutil.MakeTestEncodingConfig()
//	addrs := simtestutil.CreateIncrementalAccounts(3)
//
//
//}

func mockTxFactory(txCfg client.TxConfig) tx.Factory {
	return tx.Factory{}.
		WithTxConfig(txCfg).
		WithAccountNumber(50).
		WithSequence(23).
		WithFees("50stake").
		WithMemo("memo").
		WithChainID("test-chain")
}

func newTestTxConfig() (client.TxConfig, codec.Codec) {
	encodingConfig := moduletestutil.MakeTestEncodingConfig()
	return authtx.NewTxConfig(codec.NewProtoCodec(encodingConfig.InterfaceRegistry), authtx.DefaultSignModes), encodingConfig.Codec
}

func TestSigning(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	keyName := "val"
	mnemonic, err := createMnemonic()
	require.NoError(t, err)
	require.NotEmpty(t, mnemonic)

	//
	//// get config dir
	nodeDir := app.DefaultNodeHome + "/config"
	tmpDir, err := os.MkdirTemp(nodeDir, "test-")
	require.NoError(t, err)
	//
	kb, err := keyring.New(keyringAppName, keyring.BackendTest, tmpDir, nil, encCfg.Codec)
	require.NoError(t, err)
	//
	keyringAlgos, _ := kb.SupportedAlgorithms()
	algo, err := keyring.NewSigningAlgoFromString(string(hd.Secp256k1Type), keyringAlgos)
	require.NoError(t, err)
	//
	info, err := kb.NewAccount(keyName, mnemonic, "", sdk.FullFundraiserPath, algo)
	require.NoError(t, err)

	fmt.Println(kb)
	fmt.Println(mnemonic)
	fmt.Println(info.String())
	fmt.Println(tmpDir)
	//
	addr, err := info.GetAddress()
	require.NotEmpty(t, addr)
	require.NoError(t, err)

	factory := tx.Factory{}.
		WithTxConfig(encCfg.TxConfig).
		WithKeybase(kb).
		WithChainID("test").
		WithAccountNumber(0).
		WithSequence(0).
		WithFees("50stake")
	//

	require.NotEmpty(t, factory)

	msg1 := banktypes.NewMsgSend(addr, sdk.AccAddress("to"), nil)
	txBuilder, err := factory.BuildUnsignedTx(msg1)
	require.NoError(t, err)
	require.NotNil(t, txBuilder)
	sigs, err := txBuilder.GetTx().(signing.SigVerifiableTx).GetSignaturesV2()

	require.NoError(t, err)
	require.Empty(t, sigs)

	//
	clientCtx := client.Context{}
	//acc, err := accRet.GetAccount(clientCtx, addr)

	////
	//require.NoError(t, err)
	//
	////
	//txBuilder, err := factory.BuildUnsignedTx(msg1)
	//require.NoError(t, err)
	////
	err = authclient.SignTx(factory, clientCtx, keyName, txBuilder, true, true)
	tx := txBuilder.GetTx()
	require.NotEmpty(t, tx)
	signers, err := tx.GetSigners()
	require.NoError(t, err)
	fmt.Printf("This is the tx messages: %v\n", tx.GetMsgs())
	fmt.Printf("This is the tx signers: %v\n", signers)
	fmt.Printf("This is the gas: %v\n", tx.GetGas())
	fmt.Printf("This is the fee: %v\n", tx.GetFee())
	err = tx.ValidateBasic()
	require.NoError(t, err)
	//pubkey, err := tx.GetPubKeys()
	//require.NotEmpty(t, pubkey)
	//require.NoError(t, err)
	//
	//require.Equal(t, tx.GetMsgs()[0], msg1)
	//require.Equal(t, pubkey[0], info.PubKey)
}

func createMnemonic() (string, error) {
	entropySeed, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}

	mnemonic, err := bip39.NewMnemonic(entropySeed)
	if err != nil {
		return "", err
	}

	return mnemonic, nil
}
