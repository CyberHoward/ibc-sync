use std::{str::FromStr, fmt::Display};

use cosmrs::{rpc::{HttpClient, Client, endpoint::abci_query::AbciQuery, Url}};
use cw_orch_interchain::prelude::networks::JUNO_1;
use cw_orch_proto::tokenfactory::FUTURE_TIMEOUT_IN_NANO_SECONDS;
use ibc_chain_registry::chain::ChainData;


use ibc_relayer_types::{core::{ics24_host::{path::CommitmentsPath, identifier::{PortId, ChannelId}, IBC_QUERY_PATH}, ics04_channel::{msgs::{PacketMsg, recv_packet::MsgRecvPacket}, packet::Packet, timeout::TimeoutHeight}, ics23_commitment::{merkle::convert_tm_to_ics_merkle_proof, commitment::CommitmentProofBytes}}, proofs::Proofs, Height, signer::Signer, timestamp::Timestamp};

use crate::{packet_query::PacketIdentification, SIGNER};


async fn query(query: impl Display, rpc: String) -> anyhow::Result<AbciQuery>{

    let client = HttpClient::new(rpc.as_str())?;

    let response = client.abci_query(Some(IBC_QUERY_PATH.to_string()), query.to_string(), None, true).await?;

    Ok(response)

}

pub async fn get_packet_info(packet: PacketIdentification, network: ChainData) -> anyhow::Result<MsgRecvPacket> {
    
    // For the packet, we gtry to build the packet msg

    let commitment_path = CommitmentsPath{
        port_id: PortId::from_str(&packet.port_id)?,
        channel_id: ChannelId::from_str(&packet.channel_id)?,
        sequence: packet.sequence.into()
    };
    
    let response = query(commitment_path, network.apis.rpc[0].address.clone()).await?;

    let height = Height::new(0, response.clone().height.increment().into())?;

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


    let msg = MsgRecvPacket{
        packet: Packet{
            sequence: packet.sequence.into(),
            source_port: PortId::from_str(&packet.port_id)?,
            source_channel: ChannelId::from_str(&packet.channel_id)?,
            destination_port: PortId::from_str(&packet.dst_port_id)?,
            destination_channel: ChannelId::from_str(&packet.dst_channel_id)?,
            data: packet.data,
            timeout_height: TimeoutHeight::Never{

            },
            timeout_timestamp: Timestamp::from_nanoseconds(FUTURE_TIMEOUT_IN_NANO_SECONDS)?,
        },
        proofs,
        signer: Signer::dummy()
    };

    Ok(msg)
}
