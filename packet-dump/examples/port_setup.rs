use cosmwasm_std::coin;
use cw_orch::tokio::runtime::Runtime;
use cw_orch_interchain::prelude::{ChannelCreator, Starship};
use packet_dump::create_packets;

pub const SRC_CHAIN: &str = "stargaze-1";
pub const DST_CHAIN: &str = "juno-1";

fn setup() -> anyhow::Result<()> {
    let rt = Runtime::new()?;

    let starship = Starship::new(rt.handle().clone(), None)?;

    let funds = coin(5_000, "ustars");

    create_packets::setup(&rt, &starship.interchain_env(), SRC_CHAIN, DST_CHAIN, funds)?;
    Ok(())
}

fn main() {
    env_logger::init();
    setup().unwrap()
}
