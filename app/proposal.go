package app

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	nstypes "github.com/fatal-fruit/ns/types"
)

type BidProvider interface {
	GetMatchingBid(ctx sdk.Context, bid *nstypes.MsgBid) sdk.Tx
}

type ProposalHandler struct {
	app         App
	logger      log.Logger
	bidProvider BidProvider
}
type SignerInfo struct {
	keyName    string
	keyringDir string
}

type AppBidProvider struct {
	logger     log.Logger
	codec      codec.Codec
	signerInfo SignerInfo
	txConfig   client.TxConfig
	acctKeeper authkeeper.AccountKeeper
}

func (b *AppBidProvider) GetMatchingBid(ctx sdk.Context, bid *nstypes.MsgBid) sdk.Tx {
	//
	keyName := b.signerInfo.keyName
	nodeDir := b.signerInfo.keyringDir
	kb, err := keyring.New(AppName, keyring.BackendTest, nodeDir, nil, b.codec)

	if err != nil {
		b.logger.Error(fmt.Sprintf("Error creating keyring: %v", err))

		return nil
	}
	b.logger.Info(fmt.Sprintf("This is the keyname: %v", keyName))
	b.logger.Info(fmt.Sprintf("This is the key dir: %v", nodeDir))

	addrBz, err := kb.LookupAddressByKeyName(keyName)

	if err != nil {
		b.logger.Error(fmt.Sprintf("Error retrieving address by key name: %v", err))

		return nil
	}
	b.logger.Info(fmt.Sprintf("This is the addr  bytes: %v", addrBz))

	addCodec := address.Bech32Codec{
		Bech32Prefix: sdk.GetConfig().GetBech32AccountAddrPrefix(),
	}

	addrStr, err := addCodec.BytesToString(addrBz)
	if err != nil {
		b.logger.Error(fmt.Sprintf("Error converting bytes to str: %v", err))

		return nil
	}

	b.logger.Info(fmt.Sprintf("This is the addr string: %v", addrStr))

	sdkAddr, err := sdk.AccAddressFromBech32(addrStr)
	if err != nil {
		b.logger.Error(fmt.Sprintf("Error getting acct address from addr string: %v", err))

		return nil
	}

	b.logger.Info(fmt.Sprintf("This is the sdk addr: %v", sdkAddr.String()))

	acct := b.acctKeeper.GetAccount(ctx, sdkAddr)
	b.logger.Info(fmt.Sprintf("This is the addr acct addr: %v", acct.GetAddress()))
	b.logger.Info(fmt.Sprintf("This is the addr acct seq: %v", acct.GetSequence()))
	b.logger.Info(fmt.Sprintf("This is the addr acct number: %v", acct.GetAccountNumber()))

	factory := tx.Factory{}.
		WithTxConfig(b.txConfig).
		WithKeybase(kb).
		WithChainID(ctx.ChainID()).
		WithAccountNumber(acct.GetAccountNumber()).
		WithSequence(acct.GetSequence()).
		WithFees("50uatom")

	msg := nstypes.MsgBid{
		Name:           bid.Name,
		Owner:          addrStr,
		ResolveAddress: addrStr,
		Amount:         bid.Amount.MulInt(math.NewInt(2)),
	}
	txBuilder, err := factory.BuildUnsignedTx(&msg)
	if err != nil {
		b.logger.Error(fmt.Sprintf("Error building unsigned tx: %v", err))

		return nil
	}
	clientCtx := client.Context{}

	err = authclient.SignTx(factory, clientCtx, keyName, txBuilder, true, true)
	if err != nil {
		b.logger.Error(fmt.Sprintf("Error signing tx: %v", err))

		return nil
	}
	tx := txBuilder.GetTx()
	b.logger.Info(fmt.Sprintf("This is the tx messages: %v", tx.GetMsgs()))

	return tx
}

func (h *ProposalHandler) signBidTx(bid *nstypes.MsgBid) sdk.Tx {
	txconfig := h.app.GetTxConfig()
	sdkMessages := []sdk.Msg{}
	sdkMessages = append(sdkMessages, bid)
	builder := txconfig.NewTxBuilder()
	builder.SetMsgs(sdkMessages...)
	builder.SetSignatures()
	newTx := builder.GetTx()

	return newTx
}

func (h *ProposalHandler) NewPrepareProposal() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		var proposalTxs [][]byte

		for _, txBytes := range req.Txs {
			txconfig := h.app.GetTxConfig()
			txDecoder := txconfig.TxDecoder()
			messages, err := txDecoder(txBytes)
			if err != nil {
				h.logger.Info("Error Decoding txBytes")
				return &abci.ResponsePrepareProposal{Txs: req.Txs}, err
			}
			sdkMsgs := messages.GetMsgs()

			h.logger.Info(fmt.Sprintf("This is the txMsg: %v", len(sdkMsgs)))
			for _, msg := range sdkMsgs {
				switch msg := msg.(type) {
				case *nstypes.MsgBid:
					h.logger.Info(fmt.Sprintf("MsgBid: %v", msg.String()))
					// Get matching bid from matching engine
					newTx := h.bidProvider.GetMatchingBid(ctx, msg)

					// Build & sign transaction
					// Encode transaction to add to block proposal
					encTx, err := txconfig.TxEncoder()(newTx)
					if err != nil {
						h.logger.Info(fmt.Sprintf("Error sniping bid: %v", err.Error()))
					}
					proposalTxs = append(proposalTxs, encTx)
				default:
					h.logger.Info(fmt.Sprintf("Regular tx: %v", msg.String()))
				}

			}
		}
		return &abci.ResponsePrepareProposal{Txs: proposalTxs}, nil
	}
}

func SignAuc(cdc codec.Codec) (sdk.Tx, error) {
	nodeDir := DefaultNodeHome
	keyringBackend := "test"
	keyName := "val"
	keyFile := nodeDir + "/config/keyring"
	kb, err := keyring.New(sdk.KeyringServiceName(), keyringBackend, keyFile, nil, cdc)

	addrStr, err := kb.LookupAddressByKeyName(keyName)
	sdkAddr, err := sdk.AccAddressFromBech32(string(addrStr))
	fmt.Sprintf("This is the addrString: %s", sdkAddr.String())

	if err != nil {
		return nil, err
	}
	return nil, nil
}
