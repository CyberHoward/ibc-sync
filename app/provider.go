package app

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

type BidProvider interface {
	GetMatchingBid(ctx sdk.Context, bid *nstypes.MsgBid) sdk.Tx
}

type LocalSigner struct {
	keyName    string
	keyringDir string
	codec      codec.Codec
	txConfig   client.TxConfig
	kb         keyring.Keyring
	lg         log.Logger
}
type LocalBidProvider struct {
	logger     log.Logger
	codec      codec.Codec
	signer     LocalSigner
	txConfig   client.TxConfig
	acctKeeper authkeeper.AccountKeeper
}

func (bp *LocalBidProvider) Init() error {
	return bp.signer.Init(bp.txConfig, bp.codec, bp.logger)
}

func (ls *LocalSigner) Init(txCfg client.TxConfig, cdc codec.Codec, logger log.Logger) error {
	if len(ls.keyName) == 0 || len(ls.keyringDir) == 0 {
		return fmt.Errorf("keyName and keyringDir must be set")
	}

	ls.txConfig = txCfg
	ls.codec = cdc
	ls.lg = logger

	kb, err := keyring.New(AppName, keyring.BackendTest, ls.keyringDir, nil, ls.codec)
	if err != nil {
		return err
	}
	ls.kb = kb
	return nil
}

func (ls *LocalSigner) RetreiveSigner(ctx sdk.Context, actKeeper authkeeper.AccountKeeper) (types.AccountI, error) {
	lg := ls.lg
	addrBz, err := ls.kb.LookupAddressByKeyName(ls.keyName)

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

	err = authclient.SignTx(factory, clientCtx, ls.keyName, txBuilder, true, true)
	if err != nil {
		ls.lg.Error(fmt.Sprintf("Error signing tx: %v", err))

		return nil
	}
	return txBuilder.GetTx()
}

func (b *LocalBidProvider) GetMatchingBid(ctx sdk.Context, bid *nstypes.MsgBid) sdk.Tx {
	acct, err := b.signer.RetreiveSigner(ctx, b.acctKeeper)
	if err != nil {
		b.logger.Error(fmt.Sprintf("Error retrieving signer: %v", err))
		return nil
	}

	msg := nstypes.MsgBid{
		Name:           bid.Name,
		Owner:          acct.GetAddress().String(),
		ResolveAddress: acct.GetAddress().String(),
		Amount:         bid.Amount.MulInt(math.NewInt(2)),
	}

	newTx := b.signer.BuildAndSignTx(ctx, acct, msg)
	return newTx
}
