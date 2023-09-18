package provider

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	nstypes "github.com/fatal-fruit/ns/types"
)

/*
	This implementation is for demo purposes only and does not reflect all limitations and
	constraints of a live distributed network.

	Bid Provider is an embedded solution to demonstrate an interface an application could
	leverage to extract MEV when building and proposing a block. In this example, the
	application is building and signing transactions locally for the sake of a simplicity.
	Alternatively, another implementation could instead take transactions submitted directly
	via RPC to its app side mempool, and could even implement a separate custom mempool for
	special transactions of this nature.
*/

type BidProvider interface {
	GetMatchingBid(ctx sdk.Context, bid *nstypes.MsgBid) sdk.Tx
}

type LocalSigner struct {
	KeyName    string
	KeyringDir string
	codec      codec.Codec
	txConfig   client.TxConfig
	kb         keyring.Keyring
	lg         log.Logger
}

type LocalBidProvider struct {
	Logger     log.Logger
	Codec      codec.Codec
	Signer     LocalSigner
	TxConfig   client.TxConfig
	AcctKeeper authkeeper.AccountKeeper
}

func (bp *LocalBidProvider) Init() error {
	return bp.Signer.Init(bp.TxConfig, bp.Codec, bp.Logger)
}

func (ls *LocalSigner) Init(txCfg client.TxConfig, cdc codec.Codec, logger log.Logger) error {

	if len(ls.KeyName) == 0 {
		return fmt.Errorf("keyName  must be set")
	}

	if len(ls.KeyringDir) == 0 {
		return fmt.Errorf("keyDir  must be set")

	}

	ls.txConfig = txCfg
	ls.codec = cdc
	ls.lg = logger

	kb, err := keyring.New("cosmos", keyring.BackendTest, ls.KeyringDir, nil, ls.codec)
	if err != nil {
		return err
	}
	ls.kb = kb
	return nil
}

func (ls *LocalSigner) RetreiveSigner(ctx sdk.Context, actKeeper authkeeper.AccountKeeper) (types.AccountI, error) {
	lg := ls.lg

	lg.Info(fmt.Sprintf("Keyring Dir: %v", ls.KeyringDir))
	addrBz, err := ls.kb.LookupAddressByKeyName(ls.KeyName)

	if err != nil {
		lg.Error(fmt.Sprintf("Error retrieving address by key name: %v", err))
		return nil, err
	}

	addCodec := address.Bech32Codec{
		Bech32Prefix: sdk.GetConfig().GetBech32AccountAddrPrefix(),
	}

	addrStr, err := addCodec.BytesToString(addrBz)
	if err != nil {
		lg.Error(fmt.Sprintf("Error converting address bytes to str: %v", err))
		return nil, err
	}

	sdkAddr, err := sdk.AccAddressFromBech32(addrStr)
	if err != nil {
		lg.Error(fmt.Sprintf("Error getting acct address from addr string: %v", err))
		return nil, err
	}

	acct := actKeeper.GetAccount(ctx, sdkAddr)
	return acct, nil
}

func (ls *LocalSigner) BuildAndSignTx(ctx sdk.Context, acct types.AccountI, msg nstypes.MsgBid) sdk.Tx {
	factory := tx.Factory{}.
		WithTxConfig(ls.txConfig).
		WithKeybase(ls.kb).
		WithChainID(ctx.ChainID()).
		WithAccountNumber(acct.GetAccountNumber()).
		WithSequence(acct.GetSequence()).
		WithFees("50uatom")

	txBuilder, err := factory.BuildUnsignedTx(&msg)
	if err != nil {
		ls.lg.Error(fmt.Sprintf("Error building unsigned tx: %v", err))

		return nil
	}
	clientCtx := client.Context{}

	err = authclient.SignTx(factory, clientCtx, ls.KeyName, txBuilder, true, true)
	if err != nil {
		ls.lg.Error(fmt.Sprintf("Error signing tx: %v", err))

		return nil
	}
	return txBuilder.GetTx()
}

func (b *LocalBidProvider) GetMatchingBid(ctx sdk.Context, bid *nstypes.MsgBid) sdk.Tx {
	acct, err := b.Signer.RetreiveSigner(ctx, b.AcctKeeper)
	b.Logger.Info("Retrieved Signer")
	if err != nil {
		b.Logger.Error(fmt.Sprintf("Error retrieving signer: %v", err))
		return nil
	}
	b.Logger.Info("Created new bid")

	msg := nstypes.MsgBid{
		Name:           bid.Name,
		Owner:          acct.GetAddress().String(),
		ResolveAddress: acct.GetAddress().String(),
		Amount:         bid.Amount.MulInt(math.NewInt(2)),
	}

	newTx := b.Signer.BuildAndSignTx(ctx, acct, msg)
	return newTx
}
