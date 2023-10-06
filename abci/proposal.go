package abci

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/fatal-fruit/cosmapp/mempool"
	"github.com/fatal-fruit/cosmapp/provider"
	nstypes "github.com/fatal-fruit/ns/types"
)

func NewPrepareProposalHandler(
	lg log.Logger,
	txCg client.TxConfig,
	cdc codec.Codec,
	mp *mempool.ThresholdMempool,
	pv provider.TxProvider,
	runProv bool,
) *PrepareProposalHandler {
	return &PrepareProposalHandler{
		logger:      lg,
		txConfig:    txCg,
		cdc:         cdc,
		mempool:     mp,
		txProvider:  pv,
		runProvider: runProv,
	}
}

// Takes the votes and creates the proposal with them.
// It does this by creating the special tx and appending it with the ClientUpdate msgs and Packet msgs, after which the other txs are added.
func (h *PrepareProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		h.logger.Info(fmt.Sprintf("🛠️ :: Prepare Proposal"))
		var proposalTxs [][]byte

		// Get Vote Extensions
		if req.Height > 2 {

			// Construct Special Transaction
			ve, err := processVoteExtensions(req, h.logger)
			if err != nil {
				h.logger.Error(fmt.Sprintf("❌️ :: Unable to process Vote Extensions: %v", err))
			}

			// Deserialize the json data into the appropriate structs.
			// JSON format is as follows:
			// {
			//	"height": "height of the remote chain",
			// 	"client_updates": [ "base64 encoded client update", "base64 encoded client update"]
			// 	"packet_updates": ["base64 encoded packet commitment", "base64 encoded packet commitment"]
			// }
			// 1. load the json
			// 2. unmarshal the json into the struct
			// 3. base64 decode the messages and proto deserialize them into the appropriate structs.
			jsonFile, err := os.ReadFile("/Users/robin/Programming/External/abci-workshop/data.json")
			if err != nil {
				panic(err)
			}

			// Parse the JSON contents into FetchedIbcUpdate
			var fetchedIbcUpdate FetchedIbcUpdate
			err = json.Unmarshal(jsonFile, &fetchedIbcUpdate)
			if err != nil {
				panic(err)
			}

			// Parse client update msgs
			var clientUpdateMsgs []clienttypes.MsgUpdateClient
			for _, clientUpdate := range fetchedIbcUpdate.ClientUpdates {
				// Base64 decode the messages into bytes
				h.logger.Error(fmt.Sprintf("❌️ client_update: %v", clientUpdate))
				clientBytes, err := base64.StdEncoding.DecodeString(clientUpdate)
				if err != nil {
					panic(err)
				}

				h.logger.Error(fmt.Sprintf("❌️ client_bytes: %v", clientBytes))

				var msg clienttypes.MsgUpdateClient
				err = h.cdc.Unmarshal(clientBytes, &msg)
				if err != nil {
					panic(err)
				}
				h.logger.Error(fmt.Sprintf("❌️ client_msg: %v", msg))

				clientUpdateMsgs = append(clientUpdateMsgs, msg)
			}

			// Parse packet msgs
			var packetUpdateMsgs []channeltypes.MsgRecvPacket
			for _, packetUpdate := range fetchedIbcUpdate.Packets {
				// Base64 decode the messages into bytes
				packetBytes, err := base64.StdEncoding.DecodeString(packetUpdate)
				if err != nil {
					panic(err)
				}

				var msg channeltypes.MsgRecvPacket
				err = h.cdc.Unmarshal(packetBytes, &msg)
				if err != nil {
					panic(err)
				}
				packetUpdateMsgs = append(packetUpdateMsgs, msg)
			}

			var Msgs []sdk.Msg = []sdk.Msg{&clientUpdateMsgs[0.], &packetUpdateMsgs[0.]}

			// TODO: Append Client Updates and Packet Commitments to proposal
			ibc_tx := h.txProvider.SignMsgs(ctx, Msgs)

			// Marshal Special Transaction
			bz, err := json.Marshal(ve)
			if err != nil {
				h.logger.Error(fmt.Sprintf("❌️ :: Unable to marshal Vote Extensions: %v", err))
			}

			// Marshal IBC txs
			ibc_bz, err := json.Marshal(ibc_tx)
			if err != nil {
				h.logger.Error(fmt.Sprintf("❌️ :: Unable to marshal IBC txs: %v", err))
			}

			// Append Special Transaction to proposal
			proposalTxs = append(proposalTxs, bz)
			proposalTxs = append(proposalTxs, ibc_bz)

		}

		// add txs to front of block
		var txs []sdk.Tx
		itr := h.mempool.Select(context.Background(), nil)
		for itr != nil {
			tmptx := itr.Tx()

			txs = append(txs, tmptx)
			itr = itr.Next()
		}
		h.logger.Info(fmt.Sprintf("🛠️ :: Number of Transactions available from mempool: %v", len(txs)))

		if h.runProvider {
			tmpMsgs, err := h.txProvider.BuildProposal(ctx, txs)
			if err != nil {
				h.logger.Error(fmt.Sprintf("❌️ :: Error Building Custom Proposal: %v", err))
			}
			txs = tmpMsgs
		}

		for _, sdkTxs := range txs {
			txBytes, err := h.txConfig.TxEncoder()(sdkTxs)
			if err != nil {
				h.logger.Info(fmt.Sprintf("❌~Error encoding transaction: %v", err.Error()))
			}
			proposalTxs = append(proposalTxs, txBytes)
		}

		h.logger.Info(fmt.Sprintf("🛠️ :: Number of Transactions in proposal: %v", len(proposalTxs)))

		return &abci.ResponsePrepareProposal{Txs: proposalTxs}, nil
	}
}

// Verifies that the updates available in the votes are included in the block proposal. If not, the proposal is rejected.
func (h *ProcessProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (resp *abci.ResponseProcessProposal, err error) {
		h.Logger.Info(fmt.Sprintf("⚙️ :: Process Proposal"))

		// The first transaction will always be the Special Transaction.
		// It includes the height of the remote chain.
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
			// If some client updates are
			// if st.NrClientUpdates > 0 {
			// 	h.Logger.Info(fmt.Sprintf("⚙️:: There are updates in the Special Transaction"))

			// TODO: Validate that the txs that were included in this proposal are actually the ones that were voted for.

			// var updates []
			// for i, b := range st.Bids {
			// 	var bid nstypes.MsgBid
			// 	h.Codec.Unmarshal(b, &bid)
			// 	h.Logger.Info(fmt.Sprintf("⚙️:: Special Transaction Bid No %v :: %v", i, bid))
			// 	bids = append(bids, bid)
			// }
			// Validate Client updates and Packet commitments in Tx
			// txs := req.Txs[1:]
			// ok, err := ValidateBids(h.TxConfig, bids, txs, h.Logger)
			// if err != nil {
			// 	h.Logger.Error(fmt.Sprintf("❌️:: Error validating bids in Process Proposal :: %v", err))
			// 	return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
			// }
			// if !ok {
			// 	h.Logger.Error(fmt.Sprintf("❌️:: Unable to validate bids in Process Proposal :: %v", err))
			// 	return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
			// }
			h.Logger.Info("⚙️:: Successfully validated updates in Process Proposal")
			// }
		}

		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
	}
}

// Called when proposing a block. Checks all the votes and returns the special tx.
func processVoteExtensions(req *abci.RequestPrepareProposal, log log.Logger) (SpecialTransaction, error) {
	log.Info(fmt.Sprintf("🛠️ :: Process Vote Extensions"))

	// Create empty response
	st := SpecialTransaction{
		0,
	}

	// Get Vote Ext for H-1 from Req
	// voteExt := req.GetLocalLastCommit()
	// votes := voteExt.Votes

	// // Iterate through votes
	// var ve AppVoteExtension
	// var ibcSync IbcUpdate
	// for _, vote := range votes {
	// 	// Unmarshal to AppExt
	// 	log.Error(fmt.Sprintf("❌ :: vote: %v", vote))

	// 	err := json.Unmarshal(vote.VoteExtension, &ve)
	// 	if err != nil {
	// 		log.Error(fmt.Sprintf("❌ :: Error unmarshalling Vote Extensions"))
	// 	}

	// 	// if len(ve.IbcUpdate) != 0 {
	// 	// TODO: Collect all votes and determine which packets to commit.
	// 	// For now we just use the first vote and commit that.
	// 	err2 := json.Unmarshal(ve.IbcUpdate, &ibcSync)

	// 	if err2 != nil {
	// 		log.Error(fmt.Sprintf("❌ :: Error unmarshalling Vote Extension"))
	// 	}

	// 	// set all the fields
	// 	st.Height = int(ve.Height)
	// 	// }

	// 	// If Bids in VE, append to Special Transaction
	// 	// if len(ve.Bids) > 0 {
	// 	// 	log.Info("🛠️ :: Bids in VE")
	// 	// 	for _, b := range ve.Bids {
	// 	// 		st.Bids = append(st.Bids, b)
	// 	// 	}
	// 	// }
	// }

	return st, nil
}

// func ValidateBids(txConfig client.TxConfig, veBids []nstypes.MsgBid, proposalTxs [][]byte, logger log.Logger) (bool, error) {
// 	var proposalBids []*nstypes.MsgBid
// 	for _, txBytes := range proposalTxs {
// 		txDecoder := txConfig.TxDecoder()
// 		messages, err := txDecoder(txBytes)
// 		if err != nil {
// 			logger.Error(fmt.Sprintf("❌️:: Unable to decode proposal transactions :: %v", err))

// 			return false, err
// 		}
// 		sdkMsgs := messages.GetMsgs()
// 		for _, m := range sdkMsgs {
// 			switch m := m.(type) {
// 			case *nstypes.MsgBid:
// 				proposalBids = append(proposalBids, m)
// 			}
// 		}
// 	}

// 	bidFreq := make(map[string]int)
// 	totalVotes := len(veBids)
// 	for _, b := range veBids {
// 		h, err := Hash(&b)
// 		if err != nil {
// 			logger.Error(fmt.Sprintf("❌️:: Unable to produce bid frequency map :: %v", err))

// 			return false, err
// 		}
// 		bidFreq[h]++
// 	}

// 	thresholdCount := int(float64(totalVotes) * 0.5)
// 	logger.Info(fmt.Sprintf("🛠️ :: VE Threshold: %v", thresholdCount))
// 	ok := true
// 	logger.Info(fmt.Sprintf("🛠️ :: Number of Proposal Bids: %v", len(proposalBids)))

// 	for _, p := range proposalBids {

// 		key, err := Hash(p)
// 		if err != nil {
// 			logger.Error(fmt.Sprintf("❌️:: Unable to hash proposal bid :: %v", err))

// 			return false, err
// 		}
// 		freq := bidFreq[key]
// 		logger.Info(fmt.Sprintf("🛠️ :: Frequency for Proposal Bid: %v", freq))
// 		if freq < thresholdCount {
// 			logger.Error(fmt.Sprintf("❌️:: Detected invalid proposal bid :: %v", p))

// 			ok = false
// 		}
// 	}
// 	return ok, nil
// }

func Hash(m *nstypes.MsgBid) (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
