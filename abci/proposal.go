package abci

import (
	"cosmossdk.io/log"
	"encoding/json"
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

//type InjectedVoteExt struct {
//	VoteExtSigner []byte
//	Bids          []*nstypes.MsgBid
//}

type InjectedVoteExt struct {
	Signer  string
	Message string
}

type InjectedVotes struct {
	Votes []InjectedVoteExt
}

func (h *ProposalHandler) NewPrepareProposal() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		var proposalTxs [][]byte

		if req.Height > 1 {
			voteExt, err := processVoteExtensions(req, h.Logger)
			if err != nil {
				h.Logger.Error(fmt.Sprintf("Unable to process Vote Extensions: %w", err))
			} else if voteExt != nil {
				var testVts InjectedVotes
				err = json.Unmarshal(voteExt, &testVts)
				h.Logger.Info(fmt.Sprintf("🛠️~These are the injected Vote Extensions: %v", testVts.Votes))
				proposalTxs = append(proposalTxs, voteExt)
			}
		}

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

func (h *ProcessProposalHandler) NewProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (resp *abci.ResponseProcessProposal, err error) {
		bdCnt := map[string]int{}
		h.Logger.Info(fmt.Sprintf("⚙️~Processing Proposal for height : %v", req.Height))
		var votes InjectedVotes

		if len(req.Txs) >= 1 {
			if err := json.Unmarshal(req.Txs[0], &votes); err != nil {
				h.Logger.Error(fmt.Sprintf("⚙️~Error Unmarshalling Vote Extensions : %w", err))

				return &abci.ResponseProcessProposal{abci.ResponseProcessProposal_REJECT}, err
			}

			//for i, v := range votes.Votes {
			//	h.Logger.Info(fmt.Sprintf("⚙️ Signer for Vote Extension %v: %v", i, v.VoteExtSigner))
			//	for j, b := range v.Bids {
			//		h.Logger.Info(fmt.Sprintf("⚙️ Bid for Vote Extension %i: %v", j, b.String()))
			//		k := b.Name + b.Owner + b.ResolveAddress
			//		bdCnt[k] += 1
			//	}
			//}

			for k, v := range bdCnt {
				h.Logger.Info(fmt.Sprintf("Bid %v :: %v", k, v))
			}
		}

		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
	}
}

func processVoteExtensions(req *abci.RequestPrepareProposal, log log.Logger) ([]byte, error) {
	proposer := sdk.ConsAddress(req.ProposerAddress).String()

	log.Info("**************************************************************")
	log.Info(fmt.Sprintf("🛠️~This Validator %v is proposing transactions at height : %v", proposer, req.Height))
	log.Info("**************************************************************")

	if req.Height >= 2 {
		injV := InjectedVotes{}

		ve := req.GetLocalLastCommit()
		votes := ve.GetVotes()
		for i, v := range votes {

			val := (sdk.ConsAddress)(v.Validator.GetAddress()).String()
			var veb AppVoteExtension
			bz := v.VoteExtension
			err := json.Unmarshal(bz, &veb)
			if err != nil {
				log.Info(fmt.Sprintf("Error unmarhsalling VE tx: %w", err))
			}

			log.Info(fmt.Sprintf("🗳️~Vote extensions %v at height %v for validator :: %v", i, veb.Height, val))

			//for j, b := range veb.Bids {
			//	log.Info(fmt.Sprintf("🗳️~Bid %v of vote extension: %v", j, b.String()))
			//}

			//inj := InjectedVoteExt{
			//	VoteExtSigner: v.ExtensionSignature,
			//	Bids:          veb.Bids,
			//}

			inj := InjectedVoteExt{
				Signer:  val,
				Message: veb.Message,
			}
			injV.Votes = append(injV.Votes, inj)
		}

		injectedBz, err := json.Marshal(injV)
		if err != nil {
			log.Info(fmt.Sprintf("Error marhsalling VE tx: %w", err))
		}

		return injectedBz, nil
	}
	return nil, nil
}
