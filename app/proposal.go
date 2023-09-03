package app

import (
	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ProposalHandler struct {
	app    App
	logger log.Logger
}

func (h *ProposalHandler) NewPrepareProposal() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		var proposalTxs [][]byte

		for _, tx := range req.Txs {

			proposalTxs = append(proposalTxs, tx)
		}

		return &abci.ResponsePrepareProposal{
			Txs: proposalTxs,
		}, nil
	}
}
