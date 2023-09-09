package app

import (
	"cosmossdk.io/log"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	nstypes "github.com/fatal-fruit/ns/types"
)

type ProposalHandler struct {
	app         App
	logger      log.Logger
	bidProvider BidProvider
}

func (h *ProposalHandler) NewPrepareProposal() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		var proposalTxs [][]byte

		for _, txBytes := range req.Txs {
			txconfig := h.app.GetTxConfig()
			txDecoder := txconfig.TxDecoder()
			messages, err := txDecoder(txBytes)
			if err != nil {
				h.logger.Error("Error Decoding txBytes")
				return &abci.ResponsePrepareProposal{Txs: req.Txs}, err
			}
			sdkMsgs := messages.GetMsgs()

			var updatedTx []byte
			for _, msg := range sdkMsgs {
				switch msg := msg.(type) {
				case *nstypes.MsgBid:
					// Get matching bid from matching engine
					newTx := h.bidProvider.GetMatchingBid(ctx, msg)
					// Encode transaction to add to block proposal
					encTx, err := txconfig.TxEncoder()(newTx)
					if err != nil {
						h.logger.Info(fmt.Sprintf("Error sniping bid: %v", err.Error()))
					}

					updatedTx = encTx
				default:
				}

			}
			if updatedTx != nil {
				proposalTxs = append(proposalTxs, updatedTx)
			} else {
				proposalTxs = append(proposalTxs, txBytes)
			}
		}
		return &abci.ResponsePrepareProposal{Txs: proposalTxs}, nil
	}
}
