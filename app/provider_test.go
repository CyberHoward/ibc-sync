package app_test

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/go-bip39"
	"github.com/stretchr/testify/require"
	"testing"
)

type testFixture struct {
	ctx sdk.Context
	k   authkeeper.AccountKeeper

	addrs []sdk.AccAddress
}

func createLocalKey(t *testing.T, kb keyring.Keyring) {

	//authKeeper.EXPECT().GetAccount(suite.ctx, moduleAcc.GetAddress()).Return(moduleAcc)
	mnemonic, err := createMnemonic()
	require.NoError(t, err)
	require.NotEmpty(t, mnemonic)

	keyringAlgos, _ := kb.SupportedAlgorithms()
	algo, err := keyring.NewSigningAlgoFromString(string(hd.Secp256k1Type), keyringAlgos)
	require.NoError(t, err)

	info, err := kb.NewAccount("val", mnemonic, "", sdk.FullFundraiserPath, algo)
	fmt.Println(info)
	require.NoError(t, err)

	//addr, err := info.GetAddress()
}

func TestSigning(t *testing.T) {

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
