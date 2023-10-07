use std::{str::FromStr, thread::sleep, time::Duration};

use cosmrs::{proto::{traits::Message, ibc::core::{client::v1::MsgUpgradeClient, channel::v1::MsgRecvPacketResponse}}, Any};
use cw_orch_interchain::{tokio::runtime::{Handle, Runtime}, prelude::networks::JUNO_1, state::ChainState};
use cw_orch_interchain::interchain::{
    channel::{IbcPort, InterchainChannel},
    InterchainEnv,
};
use cw_orch_interchain::prelude::*;
use cw_orch_proto::tokenfactory::{transfer_tokens, create_transfer_channel};
use ibc_relayer_types::core::ics24_host::identifier::{ChannelId, PortId};

use crate::{packet_query::{query_all_unreceived_packets}, packet_info::get_packet_info, update_client::get_client_message, ibc::Ibc};
use futures::future::try_join_all;
use ibc_proto::ibc::core::{channel::v1::MsgRecvPacket as RawMsgRecvPacket, client::v1::MsgUpdateClientResponse};
use base64::{engine::general_purpose, Engine as _};
use serde::Serialize;

#[derive(Serialize)]
pub struct ValidatorProto{
    pub height: u64,
    pub client_updates: Vec<String>,
    // pub client_updates: Vec<UpdateClientMe>,
    pub packets: Vec<String>,
}

#[derive(Serialize)]
pub struct UpdateClientMe{
    client_id: String,
    client_message_data: Vec<u8>,
    signer: String
}


pub fn setup<IBC: InterchainEnv<Daemon>>(
    rt: &Runtime,
    interchain: &IBC,
    src_chain_id: &str,
    dst_chain_id: &str,
    funds: Coin,
) -> anyhow::Result<ValidatorProto> {
    let src_chain = interchain.chain(src_chain_id)?;
    let dst_chain = interchain.chain(dst_chain_id)?;

    // Now create channel between 2 chains
    // create_ibc_trail(
    //     rt,
    //     interchain,
    //     src_chain_id,
    //     dst_chain_id,
    //     &src_chain,
    //     &dst_chain,
    //     funds,
    // )?;
    // Now see what packets are still in transit
    let packets_to_relay = rt.block_on(query_all_unreceived_packets(
        src_chain.channel(),
        dst_chain.channel(),
        dst_chain_id,
    ))?;

    log::info!("{:} packets to relay", packets_to_relay.len());

    // We create all the messages we need to relay packets
    let packet_msgs = rt.block_on(try_join_all(
        packets_to_relay.iter().map(|p| async{
            let recv_packet = get_packet_info(p.clone(), src_chain.clone(), dst_chain.clone()).await?;
            
            let raw_recv_packet = RawMsgRecvPacket::from(recv_packet);


            Ok::<_, anyhow::Error>(general_purpose::STANDARD.encode(raw_recv_packet.encode_to_vec()))
        })
    ))?;
    
    let update_msgs = rt.block_on(try_join_all(
        packets_to_relay.iter().map(|p| async{
            let update_client = get_client_message(p.clone(), &src_chain, &dst_chain).await?;
            
            let raw_update_client : ibc_proto::ibc::core::client::v1::MsgUpdateClient = update_client.into();

            // let update_msg = UpdateClientMe{
            //     client_id: raw_update_client.client_id,
            //     signer: raw_update_client.signer,
            //     client_message_data: raw_update_client.header.unwrap().value,
            // };

            Ok::<_, anyhow::Error>(general_purpose::STANDARD.encode(raw_update_client.encode_to_vec()))
            // Ok::<_, anyhow::Error>(update_msg)
        })
    ))?;


    // Try to broadcast a update msg

    let first_packet = rt.block_on(
        get_packet_info(packets_to_relay[0].clone(), src_chain.clone(), dst_chain.clone())
    )?;
    let raw_packet_msg : ibc_proto::ibc::core::channel::v1::MsgRecvPacket = first_packet.clone().into();

    let first_update = rt.block_on(
        get_client_message(packets_to_relay[0].clone(), &src_chain, &dst_chain)
    )?;
    let raw_update_client : ibc_proto::ibc::core::client::v1::MsgUpdateClient = first_update.into();
    log::info!("Test Updating Client");


    let resp = dst_chain.commit_any::<MsgUpdateClientResponse>(vec![
        Any{
            value:raw_update_client.encode_to_vec(), 
            type_url: "/ibc.core.client.v1.MsgUpdateClient".to_string()
        }
    ], None)?;

    log::info!("Test Updating Client succeeded");

    // let resp = dst_chain.commit_any::<MsgRecvPacketResponse>(vec![
    //     Any{
    //         value:raw_packet_msg.encode_to_vec(), 
    //         type_url: "/ibc.core.channel.v1.MsgRecvPacket".to_string()
    //     }
    // ], None)?;


    let all_update = ValidatorProto{
        height: 89,
        client_updates: update_msgs,
        packets: packet_msgs
    };

    let json_value = serde_json::to_string(&all_update)?;

    std::fs::write("all-packets.json", json_value)?;

    log::info!("All packet dumped");

    Ok(all_update)
}

pub fn create_ibc_trail<IBC: InterchainEnv<Daemon>>(
    rt: &Runtime,
    interchain: &IBC,
    src_chain_id: &str,
    dst_chain_id: &str,
    src_chain: &Daemon,
    dst_chain: &Daemon,
    funds: Coin,
) -> anyhow::Result<()> {
    let ibc_channel = rt.block_on(create_transfer_channel(src_chain_id, dst_chain_id, None, interchain))?;

    // let ibc_channel = InterchainChannel {
    //     port_a: IbcPort {
    //         chain_id: src_chain_id.to_string(),
    //         connection_id: Some("connection-0".to_string()),
    //         port: PortId::from_str("transfer")?,
    //         channel: Some(ChannelId::from_str("channel-0")?),
    //         chain: src_chain.channel(),
    //     },
    //     port_b: IbcPort {
    //         chain_id: dst_chain_id.to_string(),
    //         connection_id: Some("connection-0".to_string()),
    //         port: PortId::from_str("transfer")?,
    //         channel: Some(ChannelId::from_str("channel-0")?),
    //         chain: dst_chain.channel(),
    //     },
    // };

    let receiver = dst_chain.sender().to_string();

    // And test the transfer packet
    transfer_tokens(
        rt,
        src_chain,
        &receiver,
        &funds,
        interchain,
        &ibc_channel,
        None,
        None,
    )?;

    Ok(())
}
