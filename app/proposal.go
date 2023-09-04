package app

import (
	"cosmossdk.io/log"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	nstypes "github.com/fatal-fruit/ns/types"
)

type ProposalHandler struct {
	app    App
	logger log.Logger
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
				}
			}
			proposalTxs = append(proposalTxs, txBytes)
		}
		return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
	}
}
