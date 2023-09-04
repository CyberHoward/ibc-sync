package app

import (
	"cosmossdk.io/log"
	"encoding/json"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

type ProposalHandler struct {
	app    App
	logger log.Logger
}

func (h *ProposalHandler) NewPrepareProposal() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		var proposalTxs [][]byte

		for _, txBytes := range req.Txs {
			tx := txtypes.Tx{}

			err := json.Unmarshal(txBytes, &tx)
			if err != nil {
				return nil, err
			}

			messages := tx.GetMsgs()
			for _, msg := range messages {
				if sdk.MsgTypeURL(msg) == "Bid" {
					h.logger.Info("Bid Found")
					//	to be altered when bid added
				}
			}
			proposalTxs = append(proposalTxs, txBytes)
		}
		return nil, nil
	}
}
