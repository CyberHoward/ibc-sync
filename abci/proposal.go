package abci

import (
	"cosmossdk.io/log"
	"encoding/json"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	nstypes "github.com/fatal-fruit/ns/types"
)

type SpecialTransaction struct {
	Height int
	Bids   [][]byte
}

func (h *ProposalHandler) NewPrepareProposal() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		h.Logger.Info(fmt.Sprintf("🛠️ :: Prepare Proposal"))
		var proposalTxs [][]byte

		// Get Vote Extensions
		if req.Height > 2 {

			// Get Special Transaction
			ve, err := processVoteExtensions(req, h.Logger)
			if err != nil {
				h.Logger.Error(fmt.Sprintf("❌️ :: Unable to process Vote Extensions: %w", err))
			}

			// Marshal Special Transaction
			bz, err := json.Marshal(ve)
			if err != nil {
				h.Logger.Error(fmt.Sprintf("❌️ :: Unable to marshal Vote Extensions: %w", err))
			}

			// Append Special Transaction to proposal
			proposalTxs = append(proposalTxs, bz)
		}

		// Add Txs to Proposal
		for _, txBytes := range req.Txs {
			proposalTxs = append(proposalTxs, txBytes)

			// Artificially delay Bids -> only pull from mempool if Tx has been seen in VE
		}

		return &abci.ResponsePrepareProposal{Txs: proposalTxs}, nil
	}
}

func (h *ProcessProposalHandler) NewProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (resp *abci.ResponseProcessProposal, err error) {
		h.Logger.Info(fmt.Sprintf("⚙️ :: Process Proposal"))

		numTxs := len(req.Txs)

		if numTxs == 1 {
			h.Logger.Info(fmt.Sprintf("⚙️:: Number of transactions :: %v", numTxs))
		}

		if numTxs >= 1 {
			h.Logger.Info(fmt.Sprintf("⚙️:: Number of transactions :: %v", numTxs))
			var st SpecialTransaction
			err = json.Unmarshal(req.Txs[0], &st)
			if err != nil {
				h.Logger.Error(fmt.Sprintf("❌️:: Error unmarshalling special Tx in Process Proposal :: %w", err))
			}
			if len(st.Bids) > 0 {
				h.Logger.Info(fmt.Sprintf("⚙️:: There are bids in the Special Transaction"))
				var bids []nstypes.MsgBid
				for i, b := range st.Bids {
					var bid nstypes.MsgBid
					h.Codec.Unmarshal(b, &bid)
					h.Logger.Info(fmt.Sprintf("⚙️:: Special Transaction Bid No %v :: %v", i, bid))
					bids = append(bids, bid)
				}
				// Validate Bids in Tx
				//txs := req.Txs[1 : len(req.Txs)-1]
				// Temporarily pass req txs until fixed
				txs := req.Txs
				ok, err := validateBids(bids, txs)
				if err != nil {
					h.Logger.Error(fmt.Sprintf("❌️:: Error validating bids in Process Proposal :: %w", err))
					return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
				}
				if !ok {
					h.Logger.Error(fmt.Sprintf("❌️:: Unable to validate bids in Process Proposal :: %w", err))
					return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
				}
				h.Logger.Info("⚙️:: Successfully validated bids in Process Proposal")
			}
		}

		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
	}
}

func processVoteExtensions(req *abci.RequestPrepareProposal, log log.Logger) (SpecialTransaction, error) {
	log.Info(fmt.Sprintf("🛠️ :: Process Vote Extensions"))

	// Create empty response
	st := SpecialTransaction{
		0,
		[][]byte{},
	}

	// Get Vote Ext for H-1 from Req
	voteExt := req.GetLocalLastCommit()
	votes := voteExt.Votes

	// Iterate through votes
	var ve AppVoteExtension
	for _, vote := range votes {
		// Unmarshal to AppExt
		err := json.Unmarshal(vote.VoteExtension, &ve)
		if err != nil {
			log.Error(fmt.Sprintf("❌ :: Error unmarshalling Vote Extension"))
		}

		st.Height = int(ve.Height)

		// If Bids in VE, append to Special Transaction
		if len(ve.Bids) > 0 {
			log.Info("🛠️ :: Bids in VE")
			for _, b := range ve.Bids {
				st.Bids = append(st.Bids, b)
			}
		}
	}

	return st, nil
}

func validateBids(bids []nstypes.MsgBid, txs [][]byte) (bool, error) {

	return true, nil
}
