use cw_orch_interchain::{
    daemon::{ChainInfo, ChainKind},
    prelude::networks::osmosis::OSMO_NETWORK,
};

pub const OSMOSIS_1: ChainInfo = ChainInfo {
    kind: ChainKind::Mainnet,
    chain_id: "osmosis-1",
    gas_denom: "uosmo",
    gas_price: 0.025,
    grpc_urls: &["http://grpc.osmosis.zone:9090"],
    network_info: OSMO_NETWORK,
    lcd_url: None,
    fcd_url: None,
};
