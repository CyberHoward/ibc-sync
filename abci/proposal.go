package abci

import (
	"cosmossdk.io/log"
	"encoding/json"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	nstypes "github.com/fatal-fruit/ns/types"
)

func (h *ProposalHandler) NewPrepareProposal() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		var proposalTxs [][]byte

		h.Logger.Info(fmt.Sprintf("This is the val key :: %v", h.Keyname))
		h.Logger.Info(fmt.Sprintf("This is the number of transactions :: %v", len(req.Txs)))

		// Start checking VE at H+1 (3)
		if req.Height > 2 {
			// Process VE
			voteExt, err := processVoteExtensions(req, h.Logger)
			if err != nil {
				h.Logger.Error(fmt.Sprintf("❌~Unable to process Vote Extensions: %w", err))
			} else if len(voteExt) > 0 {
				var testVts InjectedVotes
				err = json.Unmarshal(voteExt, &testVts)
				h.Logger.Info(fmt.Sprintf("🛠️~These are the injected Vote Extensions: %v", testVts.Votes))
				for i, v := range testVts.Votes {
					//var bds []*nstypes.MsgBid
					h.Logger.Info(fmt.Sprintf("🛠️~This is Vote %v", i))
					if (v.Bids != nil) && (len(v.Bids) > 0) {
						for j, b := range v.Bids {
							var bd nstypes.MsgBid
							h.Logger.Info(fmt.Sprintf("🛠️~Bids are in VE: %v", b))
							err := h.Codec.Unmarshal(b, &bd)
							if err != nil {
								h.Logger.Error(fmt.Sprintf("❌️~Error unmarshalling vote extension %v ::  %w", j, err))
							} else {
								h.Logger.Info(fmt.Sprintf("🛠️~ Bid number %v :: %v", j, bd.String()))
							}
						}

					}

				}
				h.Logger.Info("Found valid vote extensions, appending to Proposal")
				proposalTxs = append(proposalTxs, voteExt)
			}
			h.Logger.Info("Fineshed iterating through Votes")
		}

		h.Logger.Info("Building Proposal")

		if h.RunProvider {
			h.Logger.Info(fmt.Sprintf("This is the value of the provider: %v", h.RunProvider))
			for _, txBytes := range req.Txs {
				txDecoder := h.TxConfig.TxDecoder()
				messages, err := txDecoder(txBytes)
				if err != nil {
					h.Logger.Error("❌~Error Decoding txBytes")
					return &abci.ResponsePrepareProposal{Txs: req.Txs}, err
				}
				sdkMsgs := messages.GetMsgs()

				var updatedTx []byte
				for _, msg := range sdkMsgs {
					switch msg := msg.(type) {
					case *nstypes.MsgBid:
						h.Logger.Info("Found a Bid to Snipe")

						// Get matching bid from matching engine
						newTx := h.BidProvider.GetMatchingBid(ctx, msg)
						// Encode transaction to add to block proposal
						encTx, err := h.TxConfig.TxEncoder()(newTx)
						if err != nil {
							h.Logger.Info(fmt.Sprintf("❌~Error sniping bid: %v", err.Error()))
						}

						updatedTx = encTx
					default:
					}

				}
				if updatedTx != nil {
					h.Logger.Info("Appended New Tx")
					proposalTxs = append(proposalTxs, updatedTx)
				} else {
					proposalTxs = append(proposalTxs, txBytes)
				}
			}
			return &abci.ResponsePrepareProposal{Txs: proposalTxs}, nil
		}

		return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
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
				h.Logger.Error(fmt.Sprintf("❌~Error Unmarshalling Vote Extensions : %w", err))

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
	//emptyVE := make([]byte, 0)
	log.Info("**************************************************************")
	log.Info(fmt.Sprintf("🛠️~This Validator %v is proposing transactions at height : %v", proposer, req.Height))
	log.Info("**************************************************************")
	injV := InjectedVotes{
		[]InjectedVoteExt{},
	}

	// Don't really need to do this check bc it will only be called at H+1
	if req.Height > 2 {

		ve := req.GetLocalLastCommit()
		// Number of votes ~= number of validators
		votes := ve.GetVotes()
		for i, v := range votes {

			val := (sdk.ConsAddress)(v.Validator.GetAddress()).String()
			var veb AppVoteExtension
			bz := v.VoteExtension
			log.Info(fmt.Sprintf("Length of VE: %v", len(bz)))
			err := json.Unmarshal(bz, &veb)
			if err != nil {
				// TODO: Fix
				log.Info(fmt.Sprintf("❌~Error unmarhsalling VE tx: %w", err))
			}

			log.Info(fmt.Sprintf("🗳️~Vote extensions %v at height %v for validator :: %v", i, veb.Height, val))

			//for j, b := range veb.Bids {
			//	log.Info(fmt.Sprintf("🗳️~Bid %v of vote extension: %v", j, b.String()))
			//}

			inj := InjectedVoteExt{
				VoteExtSigner: v.ExtensionSignature,
				Bids:          veb.Bids,
			}

			//inj := InjectedVoteExt{
			//	Signer:  val,
			//	Message: veb.Message,
			//}
			injV.Votes = append(injV.Votes, inj)
		}

	}
	injectedBz, err := json.Marshal(injV)
	if err != nil {
		log.Info(fmt.Sprintf("❌~Error marhsalling VE tx: %w", err))
	}

	return injectedBz, nil
	//emt, err := json.Marshal(emptyVE)
	//if err != nil {
	//	log.Error(fmt.Sprintf("❌~Error marshalling empty VE: %w", err))
	//}
	//// return empty bz
	//return emt, nil
}
