# Release Notes

### Release versioning scheme

[Gravity-Bridge/umee](https://github.com/umee-network/Gravity-Bridge) is a fork of [Althea Gravity-Bridge](https://github.com/Gravity-Bridge/Gravity-Bridge).
Both repositories don't follow semantic versioning.
The version format of gravity bridge module in this repository is `module/vX.Y.X-umee-V`, where:

- `-umee` identifies that this is umee fork version
- `vX.Y.Z` is the closest release in the upstream (Althea)
- `V` is the next consecutive number related to Umee gravity bridge fork.

## module/v1.5.3-umee-1

This release follows v1.4.1-umee-1 updates, but in v1.5 Althea line.

The main update is the migration to Cosmos SDK v0.46.x

Due to Ethereum PoS migration and potential, messy PoW fork we disable all bridge transfers
(in both directions). More specifically, the following messages are unavailable:
`MsgSendToEth`, `MsgRequestBatch`, `MsgConfirmBatch`, `MsgConfirmLogicCall`,
`MsgSendToCosmosClaim`, `MsgERC20DeployedClaim`, `MsgLogicCallExecutedClaim`.
`BatchSendToEthClaim`,

We take this action to mitigate potential risks and double spend attacks.
Once validators will catch up with Etheruem PoS fork we will discuss the following ways to re-enable
token bridge. We are considering the following options:

**NOTE**
`ValsetUpdateClaim` is active, and validators must update the validator set in Ethereum smart contract to fully enable Gravity Bridge and not get slashed.
