package abci

import (
	"context"
	"cosmossdk.io/log"
	"encoding/json"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	nstypes "github.com/fatal-fruit/ns/types"
)

type AppVoteExtension struct {
	Height int64
	Bids   []*nstypes.MsgBid
}

type VoteExtHandler struct {
	logger       log.Logger
	currentBlock int64
	mempool      sdkmempool.Mempool
}

func NewVoteExtensionHandler(lg log.Logger, mp sdkmempool.Mempool) *VoteExtHandler {
	return &VoteExtHandler{
		logger:  lg,
		mempool: mp,
	}
}

func (h *VoteExtHandler) ExtendVoteHandler() sdk.ExtendVoteHandler {
	return func(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
		h.currentBlock = req.Height
		h.logger.Info(fmt.Sprintf("Extending votes at block height : %v", req.Height))

		bids := []*nstypes.MsgBid{}

		itr := h.mempool.Select(context.Background(), nil)
		for itr != nil {
			tmptx := itr.Tx()
			sdkMsgs := tmptx.GetMsgs()

			for _, msg := range sdkMsgs {
				switch msg := msg.(type) {
				case *nstypes.MsgBid:
					// Get matching bid from matching engine
					//encTx, err := h.TxConfig.TxEncoder()(newTx)
					bids = append(bids, msg)
				default:
				}
			}
			h.mempool.Remove(tmptx)
			itr = itr.Next()
		}

		voteExt := AppVoteExtension{
			Height: req.Height,
			Bids:   bids,
		}

		bz, err := json.Marshal(voteExt)
		if err != nil {
			return nil, fmt.Errorf("Error marshalling VE: %w", err)
		}

		return &abci.ResponseExtendVote{VoteExtension: bz}, nil
	}
}
