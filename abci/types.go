package abci

import (
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/fatal-fruit/cosmapp/provider"
)

type ProposalHandler struct {
	TxConfig    client.TxConfig
	Logger      log.Logger
	BidProvider provider.BidProvider
	Codec       codec.Codec
	Keyname     string
	RunProvider bool
}

type ProcessProposalHandler struct {
	TxConfig client.TxConfig
	Codec    codec.Codec
	Logger   log.Logger
}

type VoteExtHandler struct {
	logger       log.Logger
	currentBlock int64
	mempool      sdkmempool.Mempool
	cdc          codec.Codec
}

type InjectedVoteExt struct {
	VoteExtSigner []byte
	Bids          [][]byte
}

type InjectedVotes struct {
	Votes []InjectedVoteExt
}

type AppVoteExtension struct {
	Height int64
	Bids   [][]byte
}

type SpecialTransaction struct {
	Height int
	Bids   [][]byte
}
