package v3

import (
	"github.com/Gravity-Bridge/Gravity-Bridge/module/x/gravity/types"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

func MigrateFundsToDrainAccount(ctx sdk.Context, storeKey storetypes.StoreKey, ak *authkeeper.AccountKeeper, bk *bankkeeper.BaseKeeper) error {
	ctx.Logger().Info("Shutdown Upgrade: Enter MigrateFundsToDrainAccount()")

	// First we set the drain account
	drainAcc := sdk.MustAccAddressFromBech32("umee1uuwjqrgyphm4ac20dufs7dyz0rjl3un49jg8xe")
	ctx.KVStore(storeKey).Set(types.DrainAccKey, drainAcc.Bytes())

	modAcc := ak.GetModuleAccount(ctx, types.ModuleName).GetAddress()
	balances := bk.GetAllBalances(ctx, modAcc)
	if balances.Empty() {
		return nil
	}

	return bk.SendCoinsFromModuleToAccount(ctx, types.ModuleName, drainAcc, balances)
}
