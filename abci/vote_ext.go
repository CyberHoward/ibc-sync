package abci

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/fatal-fruit/cosmapp/mempool"
)

func NewVoteExtensionHandler(lg log.Logger, mp *mempool.ThresholdMempool, cdc codec.Codec) *VoteExtHandler {
	return &VoteExtHandler{
		logger:  lg,
		mempool: mp,
		cdc:     cdc,
	}
}

// Here we extend the vote. We do this by reading a local json that contains any packets that require relaying.
func (h *VoteExtHandler) ExtendVoteHandler() sdk.ExtendVoteHandler {
	return func(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
		h.logger.Info(fmt.Sprintf("Extending votes at block height : %v", req.Height))

		ibcUpdate := IbcUpdate{
			Packet:       channeltypes.MsgRecvPacket{},
			ClientUpdate: clienttypes.MsgUpdateClient{},
		}

		// This is the packet and client update
		voteExtIbcUpdate, err := json.Marshal(ibcUpdate)

		if err != nil {
			return nil, fmt.Errorf("Error marshalling IbcUpdate: %w", err)
		}

		// TODO: Fetch IBC data with Go here. For now, we just use the local json file.

		// Create vote extension
		voteExt := AppVoteExtension{
			Height:    req.Height,
			IbcUpdate: voteExtIbcUpdate,
		}

		// Encode Vote Extension
		bz, err := json.Marshal(voteExt)
		h.logger.Error(fmt.Sprintf("❌ :: submitted vote: %v", bz))

		if err != nil {
			return nil, fmt.Errorf("Error marshalling VE: %w", err)
		}

		return &abci.ResponseExtendVote{VoteExtension: bz}, nil
	}
}
