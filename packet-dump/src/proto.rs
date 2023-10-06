#![allow(non_snake_case)]

use cosmrs::{tx::Msg, AccountId, Result, proto::ibc::applications::transfer::v1::MsgTransferResponse};

use cw_orch_interchain::interchain::{
    types::IbcTxAnalysis, IbcQueryHandler, InterchainEnv,
    InterchainError, channel::InterchainChannel,
};

use tonic::transport::Channel;

use std::str::FromStr;

use anyhow::Result as AnyResult;
use cosmrs::Denom;
use cosmwasm_std::Coin;
use cw_orch::environment::{CwEnv, TxHandler};

use ibc_relayer_types::core::ics24_host::identifier::PortId;
use cw_orch::tokio::runtime::Runtime;

use crate::ics20::MsgTransfer;
use cw_orch::prelude::FullNode;

// 1 hour should be sufficient for packet timeout
const FUTURE_TIMEOUT_IN_NANO_SECONDS: u64 = 1_696_609_219_000_000_000 + 30_600_000_000_000;

/// Ibc token transfer
/// This allows transfering token over a channel using an interchain_channel object
#[allow(clippy::too_many_arguments)]
pub fn transfer_tokens<Chain: IbcQueryHandler + FullNode, IBC: InterchainEnv<Chain>>(
    rt: &Runtime,
    origin: &Chain,
    receiver: &str,
    fund: &Coin,
    interchain_env: &IBC,
    ibc_channel: &InterchainChannel<Channel>,
    timeout: Option<u64>,
    memo: Option<String>,
) -> Result<<Chain as TxHandler>::Response, InterchainError> {
    let chain_id = origin.block_info().unwrap().chain_id;

    let (source_port, _) = ibc_channel.get_ordered_ports_from(&chain_id)?;

    let any = MsgTransfer {
        source_port: source_port.port.to_string(),
        source_channel: source_port.channel.unwrap().to_string(),
        token: Some(cosmrs::Coin {
            amount: fund.amount.u128(),
            denom: Denom::from_str(fund.denom.as_str()).unwrap(),
        }),
        sender: AccountId::from_str(origin.sender().to_string().as_str()).unwrap(),
        receiver: AccountId::from_str(receiver).unwrap(),
        timeout_height: None,
        timeout_revision: None,
        timeout_timestamp: FUTURE_TIMEOUT_IN_NANO_SECONDS,
        memo,
    }
    .to_any()
    .unwrap();

    // We send tokens using the ics20 message over the channel that is passed as an argument
    let send_tx = origin
        .commit_any::<MsgTransferResponse>(
            vec![cosmrs::Any {
                type_url: any.type_url,
                value: any.value,
            }],
            None,
        )
        .unwrap();


    Ok(send_tx)
}

/* ####################### STARSHIP specific functions ########################### */

const ICS20_CHANNEL_VERSION: &str = "ics20-1";
/// Channel creation between the transfer channels of two blockchains of a starship integration
pub async fn create_transfer_channel<Chain: IbcQueryHandler, IBC: InterchainEnv<Chain>>(
    chain1: &str,
    chain2: &str,
    src_connection_id: Option<String>,
    interchain: &IBC,
) -> AnyResult<InterchainChannel<<Chain as IbcQueryHandler>::Handler>> {
    let creation = interchain
        .create_channel(
            &chain1.to_string(),
            &chain2.to_string(),
            src_connection_id,
            &PortId::transfer(),
            &PortId::transfer(),
            ICS20_CHANNEL_VERSION,
        )
        .await
        .unwrap()
        .0;

    Ok(creation)
}
