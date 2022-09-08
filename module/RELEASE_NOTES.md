# Release Notes

## module/v1.4.3-umee

Due to Ethereum PoS migration and potential, messy PoW fork we disable all bridge transfers
(in both directions). More specifically, the following messages are unavailable:
`MsgSendToEth`, `MsgRequestBatch`, `MsgConfirmBatch`, `MsgConfirmLogicCall`,
`MsgSendToCosmosClaim`, `MsgERC20DeployedClaim`, `MsgLogicCallExecutedClaim`.
`BatchSendToEthClaim`, `ValsetUpdateClaim`.

We take this action to mitigate potential risks and double spend attacks.
Once validators will catch up with Etheruem PoS fork we will discuss the following ways to re-enable
token bridge. We are considering the following options:

- Firstly re-enable Ethereum->Umee Gravity Bridge (this only requires for validators to agree on the forked chain, and is slasheable).
- Once PoW fork risks will be well understood, re-enable Cosmos->Ethereum bridge
- Migrate to other bridge.
