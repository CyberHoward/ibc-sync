use std::str::FromStr;

use cosmrs::{proto::ibc::core::client::v1::Height, rpc::{HttpClient, Client}};
use cw_orch::prelude::{queriers::{DaemonQuerier, Node}, Daemon, TxHandler};
use ibc_chain_registry::chain::ChainData;
use ibc_relayer::light_client::Verified;
use ibc_relayer_types::{core::{ics02_client::msgs::update_client::MsgUpdateClient, ics24_host::identifier::ClientId}, signer::Signer, clients::ics07_tendermint::header::Header};
use tendermint_light_client::{components::io::{ProdIo, Io, AtHeight}, types::{PeerId, LightBlock}};
use tonic::transport::Channel;

use crate::{ibc::Ibc, packet_query::PacketIdentification};


async fn verify_light_client(network: ChainData) -> anyhow::Result<Verified<LightBlock>>{

    let client = HttpClient::new(network.apis.rpc[0].address.clone().as_ref())?;

    let node_info = client
        .status()
        .await
        .map(|s| s.node_info)?;

    let peer_id = node_info.id;

    let io = ProdIo::new(peer_id, client, None);
    let target = io.fetch_light_block(AtHeight::Highest)?;

    return Ok(Verified {
        target,
        supporting: vec![],
    });
}


async fn adjust_headers(
    height: ibc_relayer_types::Height,
    target: LightBlock,
    network: ChainData
) -> anyhow::Result<Verified<Header>>{


    let trusted_validator_set = verify_light_client(network).await?.target.validators;

    let target_header = Header {
        signed_header: target.signed_header,
        validator_set: target.validators,
        trusted_height: height,
        trusted_validator_set
    };

    Ok(Verified { target: target_header, supporting: vec![] })
}



pub async fn get_client_message(packet: PacketIdentification, channel: &Daemon, network: ChainData) -> anyhow::Result<MsgUpdateClient>{
    

    let ibc = Ibc::new(channel.channel().clone());

    let connection_end = ibc.connection_end(packet.connection_id).await?.unwrap();
    let client_id = connection_end.client_id.clone();

    let Verified { target, .. } = verify_light_client(network.clone()).await?;

    let height = Node::new(channel.channel()).block_height().await?;

 
    let Verified{target, ..} = adjust_headers(ibc_relayer_types::Height::new(0, height)?, target, network).await?;

    let msg = MsgUpdateClient{
        client_id: ClientId::from_str(&client_id)?,
        header: target.into(),
        signer: Signer::from_str(&channel.sender().to_string())?
    };

    Ok(msg)
}
