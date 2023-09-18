# ABCI++ Workshop

Joint workshop between [Cosmos SDK](https://docs.cosmos.network/main) and [CometBFT](https://cometbft.com/) for [Hackmos](https://cosmoverse.org/hackmos) at [Cosmoverse](https://cosmoverse.org/).

Supported by [Informal Systems](https://informal.systems/) and [Binary Builders](https://binary.builders/).

## Content
**Overview**

The goal of this workshop is to demonstrate to developers how they might think about leveraging ABCI++ and new features in the Eden release of the Cosmos SDK.

In part 1, we want to build a custom MEV extraction solution in preparePropsoal for a validator that submits competing bids for a perpetual nameservice auction. Then in part 2, we want to mitigate leveraging vote extensions.

**Part 1**
1. We have 1 validator that wires up a custom prepareProposal handler. It checks the transactions for bids and when it sees one, it submits one into the proposal that offers a more competitive price.

**Part 2**
1. In ExtendVote, we submit an array of Bid Transactions that are currently sitting in the mempool.
2. In ProcessProposal in the subsequent round, it's going to validate that the unconfirmed bids listed in the VE match the bids in the current proposal
### Concepts

![](./figures/diagram.png)

### Part 1

### Part 2

## Developing

#### Start A Single Chain
```
make start-localnet

jq '.consensus.params.abci.vote_extensions_enable_height = "2"' ~/.cosmappd/config/genesis.json > output.json && mv output.json ~/.cosmappd/config/genesis.json

./build/cosmappd start
```

#### Start a 3 Validator Network

Make sure to run `make build`

```shell
./scripts/configure.sh
```

Read Logs
```shell

tail -f $HOME/cosmos/nodes/beacon/logs
tail -f $HOME/cosmos/nodes/val1/logs
tail -f $HOME/cosmos/nodes/val1/logs

```

Query a node
```shell
source ./scripts/vars.sh
$BINARY q bank balances $($BINARY keys show alice -a --home $HOME/cosmos/nodes/beacon --keyring-backend test) --home $HOME/cosmos/nodes/beacon --node "tcp://127.0.0.1:29170"
```

### Demo

> **Vote Extensions** are enabled from Height 2, so make sure not to submit transactions until H+1 has been comitted.


#### Single Node
```shell
./build/cosmappd tx ns reserve "bob.cosmos" $(./build/cosmappd keys show alice -a --keyring-backend test) 1000uatom --from $(./build/cosmappd keys show bob -a --keyring-backend test) -y
```
Query to verify the name has been reserved
```shell
./build/cosmappd q ns whois "bob.cosmos" -o json
```

#### 3 Validator Network
After submitting the following transaction, we should be able to see the proposal accepted or rejected in the logs.
```shell
$BINARY tx ns reserve "bob.cosmos" $($BINARY keys show alice -a --home $HOME/cosmos/nodes/beacon --keyring-backend test) 1000uatom --from $($BINARY keys show barbara -a --home $HOME/cosmos/nodes/beacon --keyring-backend test)  --home $HOME/cosmos/nodes/beacon --node "tcp://127.0.0.1:29170" -y
```
Query to verify the name has been reserved
```shell
$BINARY q ns whois "bob.cosmos" --node "tcp://127.0.0.1:29170" -o json
```

## Resources

Official Docs
- [CometBFT](https://docs.cometbft.com/v0.37/)
- [Cosmos SDK](https://docs.cosmos.network/main)

ABCI++
- [ACBI++ Spec: Basic Concepts](https://github.com/cometbft/cometbft/blob/main/spec/abci/abci++_basic_concepts.md#consensusblock-execution-methods) 
- [ABCI++ Spec: Application Requirements](https://github.com/cometbft/cometbft/blob/main/spec/abci/abci%2B%2B_app_requirements.md)
- [Skip's POB Article](https://ideas.skip.money/t/x-builder-the-first-sovereign-mev-module-for-protocol-owned-building/57)
- Videos
    - [Sergio from Informal: Presentation on ABCI++](https://youtube.com/watch?v=cAR57hZaJtM)
    - [Evan from Celestia: Possible Applications of ABCI++](https://www.youtube.com/watch?v=VGdIZLVYoRs)

Building Applications
- [Facu from Binary Builders: Building Cosmos SDK Modules](https://www.youtube.com/watch?v=9kK9uzwEeOE)
- [Interchain Developer Academy Playlist](https://www.youtube.com/watch?v=1_ottIKPfI4&list=PLE4J1RDdNh6sTSDLehUpp7vqvm2WuFWNU&pp=iAQB)
- [Cosmod's Basic module template](https://github.com/cosmosregistry/example)