use std::{str::FromStr, fmt::Display};

use cosmrs::{rpc::{HttpClient, Client, endpoint::abci_query::AbciQuery, Url}};
use cw_orch_interchain::{prelude::{networks::JUNO_1, queriers::DaemonQuerier, Daemon, TxHandler}, state::ChainState};
use cw_orch_proto::tokenfactory::FUTURE_TIMEOUT_IN_NANO_SECONDS;
use ibc_chain_registry::chain::ChainData;
use tonic::transport::Channel;

use ibc_relayer_types::{core::{ics24_host::{path::CommitmentsPath, identifier::{PortId, ChannelId}, IBC_QUERY_PATH}, ics04_channel::{msgs::{PacketMsg, recv_packet::MsgRecvPacket}, packet::Packet, timeout::TimeoutHeight}, ics23_commitment::{merkle::convert_tm_to_ics_merkle_proof, commitment::CommitmentProofBytes}}, proofs::Proofs, Height, signer::Signer, timestamp::Timestamp};

use crate::{packet_query::PacketIdentification, SIGNER, ibc::Ibc};


async fn query(query: impl Display, rpc: String, height: Height) -> anyhow::Result<AbciQuery>{

    let client = HttpClient::new(rpc.as_str())?;

    let response = client.abci_query(Some(IBC_QUERY_PATH.to_string()), query.to_string(), Some(tendermint_light_client::types::Height::try_from(height.revision_height())?), true).await?;

    Ok(response)

}

pub async fn get_packet_info(packet: PacketIdentification, src_chain: Daemon, dst_chain: Daemon) -> anyhow::Result<MsgRecvPacket> {
    
    // For the packet, we try to build the packet msg
    let commitment_path = CommitmentsPath{
        port_id: PortId::from_str(&packet.port_id)?,
        channel_id: ChannelId::from_str(&packet.channel_id)?,
        sequence: packet.sequence.into()
    };


    let ibc = Ibc::new(dst_chain.channel().clone());
    let connection_end = ibc.connection_end(packet.connection_id).await?.unwrap();
    let src_client_id = connection_end.client_id.clone();
    let height = last_trusted_height(src_client_id.clone(), dst_chain.channel().clone()).await?;

    let response = query(commitment_path, src_chain.state().chain_data.apis.rpc[0].address.clone(), height).await?;

    let proof = response
        .proof
        .clone()
        .map(|p| convert_tm_to_ics_merkle_proof(&p))
        .transpose()?;

    let proofs = Proofs::new(
        CommitmentProofBytes::try_from(proof.unwrap())?,
        None,
        None,
        None,
        height,
    )?;

    let data = serde_json::json!({"amount":"500","denom":"ustars","receiver":"juno1qjtcxl86z0zua2egcsz4ncff2gzlcndz5mfnnc","sender":"stars1qjtcxl86z0zua2egcsz4ncff2gzlcndzk4a4l4"});

    let msg = MsgRecvPacket{
        packet: Packet{
            sequence: packet.sequence.into(),
            source_port: PortId::from_str(&packet.port_id)?,
            source_channel: ChannelId::from_str(&packet.channel_id)?,
            destination_port: PortId::from_str(&packet.dst_port_id)?,
            destination_channel: ChannelId::from_str(&packet.dst_channel_id)?,
            data: serde_json::to_string(&data)?.as_bytes().to_vec(),
            timeout_height: TimeoutHeight::Never{

            },
            timeout_timestamp: Timestamp::from_nanoseconds(FUTURE_TIMEOUT_IN_NANO_SECONDS)?,
        },
        proofs,
        signer: Signer::from_str(dst_chain.sender().as_str())?
    };

    Ok(msg)
}



pub async fn last_trusted_height(client_id: String, chain: Channel) -> anyhow::Result<ibc_relayer_types::Height>{

    let ibc = Ibc::new(chain);

    let states = ibc.consensus_states(client_id).await?;

    let mut heights: Vec<_> = states.consensus_states.iter().map(|c| c.height.clone().unwrap()).collect();

    heights.sort_by_key(|h| h.revision_height);


    Ok(ibc_relayer_types::Height::new(heights[heights.len() - 3].revision_number,heights[heights.len() - 2].revision_height)?)
}