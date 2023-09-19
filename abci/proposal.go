package abci

import (
	"context"
	"cosmossdk.io/log"
	"encoding/base64"
	"encoding/json"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	nstypes "github.com/fatal-fruit/ns/types"
)

func (h *ProposalHandler) NewPrepareProposal() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		h.Logger.Info(fmt.Sprintf("🛠️ :: Prepare Proposal"))
		var proposalTxs [][]byte

		// Get Vote Extensions
		if req.Height > 2 {

			// Get Special Transaction
			ve, err := processVoteExtensions(req, h.Logger)
			if err != nil {
				h.Logger.Error(fmt.Sprintf("❌️ :: Unable to process Vote Extensions: %v", err))
			}

			// Marshal Special Transaction
			bz, err := json.Marshal(ve)
			if err != nil {
				h.Logger.Error(fmt.Sprintf("❌️ :: Unable to marshal Vote Extensions: %v", err))
			}

			// Append Special Transaction to proposal
			proposalTxs = append(proposalTxs, bz)
		}

		var txs []sdk.Tx
		itr := h.Mempool.Select(context.Background(), nil)
		for itr != nil {
			tmptx := itr.Tx()

			txs = append(txs, tmptx)
			itr = itr.Next()
		}
		h.Logger.Info(fmt.Sprintf("🛠️ :: Number of Transactions available from mempool: %v", len(txs)))

		if h.RunProvider {
			tmpMsgs, err := h.TxProvider.BuildProposal(ctx, txs)
			if err != nil {
				h.Logger.Error(fmt.Sprintf("❌️ :: Error Building Custom Proposal: %v", err))
			}
			txs = tmpMsgs
		}

		for _, sdkTxs := range txs {
			txBytes, err := h.TxConfig.TxEncoder()(sdkTxs)
			if err != nil {
				h.Logger.Info(fmt.Sprintf("❌~Error encoding transaction: %v", err.Error()))
			}
			proposalTxs = append(proposalTxs, txBytes)
		}

		h.Logger.Info(fmt.Sprintf("🛠️ :: Number of Transactions in proposal: %v", len(proposalTxs)))

		return &abci.ResponsePrepareProposal{Txs: proposalTxs}, nil
	}
}

func (h *ProcessProposalHandler) NewProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (resp *abci.ResponseProcessProposal, err error) {
		h.Logger.Info(fmt.Sprintf("⚙️ :: Process Proposal"))

		// The first transaction will always be the Special Transaction
		numTxs := len(req.Txs)
		if numTxs == 1 {
			h.Logger.Info(fmt.Sprintf("⚙️:: Number of transactions :: %v", numTxs))
		}

		if numTxs >= 1 {
			h.Logger.Info(fmt.Sprintf("⚙️:: Number of transactions :: %v", numTxs))
			var st SpecialTransaction
			err = json.Unmarshal(req.Txs[0], &st)
			if err != nil {
				h.Logger.Error(fmt.Sprintf("❌️:: Error unmarshalling special Tx in Process Proposal :: %v", err))
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
				txs := req.Txs[1:]
				// Temporarily pass req txs until fixed
				//txs := req.Txs
				ok, err := ValidateBids(h.TxConfig, bids, txs, h.Logger)
				if err != nil {
					h.Logger.Error(fmt.Sprintf("❌️:: Error validating bids in Process Proposal :: %v", err))
					return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
				}
				if !ok {
					h.Logger.Error(fmt.Sprintf("❌️:: Unable to validate bids in Process Proposal :: %v", err))
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

func ValidateBids(txConfig client.TxConfig, veBids []nstypes.MsgBid, proposalTxs [][]byte, logger log.Logger) (bool, error) {
	var proposalBids []*nstypes.MsgBid
	for _, txBytes := range proposalTxs {
		txDecoder := txConfig.TxDecoder()
		messages, err := txDecoder(txBytes)
		if err != nil {
			logger.Error(fmt.Sprintf("❌️:: Unable to decode proposal transactions :: %v", err))

			return false, err
		}
		sdkMsgs := messages.GetMsgs()
		for _, m := range sdkMsgs {
			switch m := m.(type) {
			case *nstypes.MsgBid:
				proposalBids = append(proposalBids, m)
			}
		}
	}

	bidFreq := make(map[string]int)
	totalVotes := len(veBids)
	for _, b := range veBids {
		h, err := Hash(&b)
		if err != nil {
			logger.Error(fmt.Sprintf("❌️:: Unable to produce bid frequency map :: %v", err))

			return false, err
		}
		bidFreq[h]++
	}

	thresholdCount := int(float64(totalVotes) * 0.5)
	logger.Info(fmt.Sprintf("🛠️ :: VE Threshold: %v", thresholdCount))
	ok := true
	logger.Info(fmt.Sprintf("🛠️ :: Number of Proposal Bids: %v", len(proposalBids)))

	for _, p := range proposalBids {

		key, err := Hash(p)
		if err != nil {
			logger.Error(fmt.Sprintf("❌️:: Unable to hash proposal bid :: %v", err))

			return false, err
		}
		freq := bidFreq[key]
		logger.Info(fmt.Sprintf("🛠️ :: Frequency for Proposal Bid: %v", freq))
		if freq < thresholdCount {
			logger.Error(fmt.Sprintf("❌️:: Detected invalid proposal bid :: %v", p))

			ok = false
		}
	}
	return ok, nil
}

func Hash(m *nstypes.MsgBid) (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
