package abci

import (
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/fatal-fruit/cosmapp/mempool"
	// "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/fatal-fruit/cosmapp/provider"
)

type PrepareProposalHandler struct {
	logger      log.Logger
	txConfig    client.TxConfig
	cdc         codec.Codec
	mempool     *mempool.ThresholdMempool
	txProvider  provider.TxProvider
	keyname     string
	runProvider bool
}

type ProcessProposalHandler struct {
	TxConfig client.TxConfig
	Codec    codec.Codec
	Logger   log.Logger
}

type VoteExtHandler struct {
	logger       log.Logger
	currentBlock int64
	mempool      *mempool.ThresholdMempool
	cdc          codec.Codec
}

// The vote extension of a validator
type AppVoteExtension struct {
	Height int64
	// Serialized IbcUpdate struct
	IbcUpdate []byte
}

type SpecialTransaction struct {
	Height int
}

// Data in the vote. Signer data will be ignored.
type IbcUpdate struct {
	Packet       channeltypes.MsgRecvPacket
	ClientUpdate clienttypes.MsgUpdateClient
}

type FetchedIbcUpdate struct {
	Height int64
	// Will be base64 decoded and marshalled using proto into []channeltypes.MsgRecvPacket
	ClientBytes string `json:"client_bytes"`
	// Will be base64 decoded and marshalled using proto into []clienttypes.MsgUpdateClient
	PacketBytes string `json:"packet_bytes"`
}

// Add txs to cometBFT mempool
//
// zero-trust by using the signatures provided with the votes.
