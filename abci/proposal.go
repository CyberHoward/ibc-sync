package abci

import (
	"cosmossdk.io/log"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/fatal-fruit/cosmapp/provider"
	nstypes "github.com/fatal-fruit/ns/types"
)

type ProposalHandler struct {
	TxConfig    client.TxConfig
	Logger      log.Logger
	BidProvider provider.BidProvider
}

type ProcessProposalHandler struct {
	TxConfig client.TxConfig
	Logger   log.Logger
}

type InjectedVoteExt struct {
	VoteExtSigner []byte
	Bids          []*nstypes.MsgBid
}
type InjectedVotes struct {
	Votes []InjectedVoteExt
}

func (h *ProposalHandler) NewPrepareProposal() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		var proposalTxs [][]byte
		h.Logger.Info(fmt.Sprintf("Proposing transactions at height : %v", req.Height))
		proposer := sdk.ConsAddress(req.ProposerAddress).String()
		h.Logger.Info(fmt.Sprintf("Proposer for height %v: %v", req.Height, proposer))

		//injV := InjectedVotes{}
		//
		//ve := req.GetLocalLastCommit()
		//votes := ve.GetVotes()
		//for i, v := range votes {
		//	h.Logger.Info(fmt.Sprintf("Vote extensions number :: %v", i))
		//
		//	h.Logger.Info(fmt.Sprintf("Vote extensions for validator :: %v", v.Validator.GetAddress()))
		//	var veb AppVoteExtension
		//	bz := v.VoteExtension
		//	json.Unmarshal(bz, &veb)
		//
		//	h.Logger.Info(fmt.Sprintf("Vote extensions for height : %v", veb.Height))
		//	for i, b := range veb.Bids {
		//		h.Logger.Info(fmt.Sprintf("Bid %v of vote extension: %v", i, b.String()))
		//	}
		//	inj := InjectedVoteExt{
		//		VoteExtSigner: v.ExtensionSignature,
		//		Bids:          veb.Bids,
		//	}
		//	injV.Votes = append(injV.Votes, inj)
		//}
		//
		//injectedBz, err := json.Marshal(injV)
		//if err != nil {
		//	h.Logger.Info(fmt.Sprintf("Error marhsalling VE tx: %w", err))
		//}
		//
		//proposalTxs = append(proposalTxs, injectedBz)

		for _, txBytes := range req.Txs {
			txDecoder := h.TxConfig.TxDecoder()
			messages, err := txDecoder(txBytes)
			if err != nil {
				h.Logger.Error("Error Decoding txBytes")
				return &abci.ResponsePrepareProposal{Txs: req.Txs}, err
			}
			sdkMsgs := messages.GetMsgs()

			var updatedTx []byte
			for _, msg := range sdkMsgs {
				switch msg := msg.(type) {
				case *nstypes.MsgBid:
					// Get matching bid from matching engine
					newTx := h.BidProvider.GetMatchingBid(ctx, msg)
					// Encode transaction to add to block proposal
					encTx, err := h.TxConfig.TxEncoder()(newTx)
					if err != nil {
						h.Logger.Info(fmt.Sprintf("Error sniping bid: %v", err.Error()))
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

type VE struct {
	Bid    string
	Signer []byte
}

func (h *ProcessProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (resp *abci.ResponseProcessProposal, err error) {
		//reqCount := map[string]int{}

		//bdCnt := map[string]int{}
		//h.Logger.Info(fmt.Sprintf("Processing Proposal for height : %v", req.Height))
		//var votes InjectedVotes
		//if err := json.Unmarshal(req.Txs[0], &votes); err != nil {
		//	return &abci.ResponseProcessProposal{abci.ResponseProcessProposal_REJECT}, err
		//}
		//
		//for i, v := range votes.Votes {
		//	h.Logger.Info(fmt.Sprintf("Signer for Vote Extension %v: %v", i, v.VoteExtSigner))
		//	for j, b := range v.Bids {
		//		h.Logger.Info(fmt.Sprintf("Bid for Vote Extension %i: %v", j, b.String()))
		//		k := b.Name + b.Owner + b.ResolveAddress
		//		bdCnt[k] += 1
		//	}
		//}

		//for k, v := range bdCnt {
		//	h.Logger.Info(fmt.Sprintf())
		//}

		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
	}
}
