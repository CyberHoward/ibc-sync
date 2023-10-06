pub use anyhow::Result as AnyResult;

pub mod consts;
pub mod ibc;
pub mod packet_query;
pub mod create_packets;
pub mod packet_info;
pub mod update_client;

pub const SIGNER: &str = "";

/// macro for constructing and performing a query on a CosmosSDK module.
#[macro_export]
macro_rules! cosmos_query {
    ($self:ident, $module:ident, $func_name:ident, $request_type:ident { $($field:ident : $value:expr),* $(,)?  }) => {
        {
        use $crate::cosmos_modules::$module::{
            query_client::QueryClient, $request_type,
        };
        let mut client = QueryClient::new($self.channel.clone());
        #[allow(clippy::redundant_field_names)]
        let request = $request_type { $($field : $value),* };
        let response = client.$func_name(request.clone()).await?.into_inner();
        ::log::trace!(
            "cosmos_query: {:?} resulted in: {:?}",
            request,
            response
        );
        response
    }
};
}

pub(crate) mod cosmos_modules {
    pub use cosmrs::proto::{
        cosmos::{
            auth::v1beta1 as auth,
            authz::v1beta1 as authz,
            bank::v1beta1 as bank,
            base::{abci::v1beta1 as abci, tendermint::v1beta1 as tendermint, v1beta1 as base},
            crisis::v1beta1 as crisis,
            distribution::v1beta1 as distribution,
            evidence::v1beta1 as evidence,
            feegrant::v1beta1 as feegrant,
            gov::v1beta1 as gov,
            mint::v1beta1 as mint,
            params::v1beta1 as params,
            slashing::v1beta1 as slashing,
            staking::v1beta1 as staking,
            tx::v1beta1 as tx,
            vesting::v1beta1 as vesting,
        },
        cosmwasm::wasm::v1 as cosmwasm,
        ibc::{
            applications::transfer::v1 as ibc_transfer,
            core::{
                channel::v1 as ibc_channel, client::v1 as ibc_client,
                connection::v1 as ibc_connection,
            },
        },
        tendermint::abci as tendermint_abci,
    };
}
