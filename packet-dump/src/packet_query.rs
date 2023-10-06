use std::collections::HashMap;

use crate::{ibc::Ibc, AnyResult};
use cosmrs::proto::ibc::core::connection::v1::IdentifiedConnection;
use cw_orch::{prelude::queriers::{DaemonQuerier, Node}, daemon::CosmTxResponse};
use tonic::transport::Channel;

pub async fn get_connections(
    channel: &Ibc,
    counterparty_chain_id: &str,
) -> AnyResult<Vec<IdentifiedConnection>> {
    // Ok(channel.open_connections(counterparty_chain_id).await?)
    Ok(channel.connections().await?)
}

#[derive(Debug, Clone)]
pub struct PacketIdentification {
    pub connection_id: String,
    pub port_id: String,
    pub channel_id: String,
    pub dst_port_id: String,
    pub dst_channel_id: String,
    pub sequence: u64,
    pub data: Vec<u8>,
}

/// Queries all packets that were sent from channel but not received on counterparty_channel
pub async fn query_all_unreceived_packets(
    src_chain: Channel,
    validator_chain: Channel,
    validator_chain_id: &str,
) -> AnyResult<Vec<PacketIdentification>> {
    let ibc = Ibc::new(src_chain);
    let validator_ibc = Ibc::new(validator_chain);

    let connections = get_connections(&ibc, validator_chain_id).await?;
    let mut all_packets = vec![];

    for conn in connections {
        // We get the chain id
        let client_state = ibc.connection_client(&conn.id.clone()).await?;

        let channels = ibc.connection_channels(conn.id.clone()).await?;
        for c in channels {
            let validator = c.counterparty.clone().unwrap();
            let all_commitments = ibc
                .packet_commitments(c.port_id.clone(), c.channel_id.clone())
                .await?;

            // Turn the all_commitments into a hashmap

            let all_commitments_hash: HashMap<_,_> = all_commitments.iter().map(|comm| (comm.sequence, comm)).collect();

            // Those have to be transmitted from the local chain to the counterparty chain
            let unreceived_packet_sequences = validator_ibc
                .unreceived_packets(
                    validator.port_id,
                    validator.channel_id,
                    all_commitments.iter().map(|c| c.sequence).collect(),
                )
                .await?
                .iter()
                .map(|s| PacketIdentification {
                    connection_id: conn.id.clone(),
                    port_id: c.port_id.clone(),
                    channel_id: c.channel_id.clone(),
                    dst_port_id: c.counterparty.clone().unwrap().port_id,
                    dst_channel_id: c.counterparty.clone().unwrap().channel_id,
                    sequence: *s,
                    data: all_commitments_hash.get(s).unwrap().data.clone()
                })
                .collect::<Vec<_>>();

            // We filter out only the ones that have been
            // We need to verify that the state of the packets

            all_packets.push(unreceived_packet_sequences);
        }
    }

    Ok(all_packets.into_iter().flatten().collect())
}

