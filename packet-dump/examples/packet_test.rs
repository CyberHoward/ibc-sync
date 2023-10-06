use cw_orch_interchain::prelude::*;
use cw_orch_interchain::{daemon::GrpcChannel, prelude::networks};
use ibc_chain_registry::chain::ChainData;

use cw_orch_interchain::tokio;
use packet_dump::consts::OSMOSIS_1;
async fn test() -> packet_dump::AnyResult<()> {
    env_logger::init();
    // We are trying to get all packets that need to be received on src_network
    let src_network: ChainData = networks::JUNO_1.into();
    // We are only checking counterpart_network for those packets
    let counterparty_network: ChainData = OSMOSIS_1.into();

    let grpc_channel_src =
        GrpcChannel::connect(&src_network.apis.grpc, &src_network.chain_id).await?;

    let grpc_channel_counterparty = GrpcChannel::connect(
        &counterparty_network.apis.grpc,
        &counterparty_network.chain_id,
    )
    .await?;

    packet_dump::packet_query::query_all_unreceived_packets(
        grpc_channel_counterparty,
        grpc_channel_src,
        &src_network.chain_id.to_string(),
    )
    .await?;

    Ok(())
}

#[tokio::main]
async fn main() {
    test().await.unwrap();
}
