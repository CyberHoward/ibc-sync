use std::str::FromStr;

use cosmrs::{rpc::{HttpClient, Client}};
use cw_orch_interchain::{prelude::{queriers::{DaemonQuerier, Node}, Daemon, TxHandler}, state::ChainState};
use ibc_chain_registry::chain::ChainData;
use ibc_relayer::light_client::Verified;
use ibc_relayer_types::{core::{ics02_client::msgs::update_client::MsgUpdateClient, ics24_host::identifier::ClientId}, signer::Signer, clients::ics07_tendermint::header::Header};
use tendermint_light_client::{components::io::{ProdIo, Io, AtHeight}, types::LightBlock, verifier};

use crate::{ibc::Ibc, packet_query::PacketIdentification, packet_info::last_trusted_height};


async fn verify_light_client(height: ibc_relayer_types::Height, network: ChainData) -> anyhow::Result<Verified<LightBlock>>{

    let client = HttpClient::new(network.apis.rpc[0].address.clone().as_ref())?;

    let node_info = client
        .status()
        .await
        .map(|s| s.node_info)?;

    let peer_id = node_info.id;

    let io = ProdIo::new(peer_id, client, None);
    // let target = io.fetch_light_block(AtHeight::At(height.into()))?;
    let target = io.fetch_light_block(AtHeight::Highest)?;

    Ok(Verified {
        target,
        supporting: vec![],
    })
}


async fn adjust_headers(
    height: ibc_relayer_types::Height,
    target: LightBlock,
    network: ChainData
) -> anyhow::Result<Verified<Header>>{


    let trusted_validator_set = verify_light_client(height, network).await?.target.validators;

    let target_header = Header {
        signed_header: target.signed_header,
        validator_set: target.validators,
        trusted_height: height,
        trusted_validator_set
    };

    Ok(Verified { target: target_header, supporting: vec![] })
}



pub async fn get_client_message(packet: PacketIdentification, src_channel: &Daemon, dst_channel: &Daemon) -> anyhow::Result<MsgUpdateClient>{
    
    let ibc = Ibc::new(dst_channel.channel().clone());
    let connection_end = ibc.connection_end(packet.connection_id).await?.unwrap();
    let src_client_id = connection_end.client_id.clone();

    let trusted_height = last_trusted_height(src_client_id.clone(), dst_channel.channel().clone()).await?;


    let src_network = src_channel.state().chain_data.clone();
    let Verified { target, .. } = verify_light_client(trusted_height, src_network.clone()).await?;

    log::info!("This is a trusted height {}", trusted_height);
    let Verified{target, ..} = adjust_headers(trusted_height, target, src_network).await?;

    let msg = MsgUpdateClient{
        client_id: ClientId::from_str(&src_client_id)?,
        header: target.into(),
        signer: Signer::from_str(dst_channel.sender().as_str())?
    };

    Ok(msg)
}
