package keeper

import (
	"github.com/Gravity-Bridge/Gravity-Bridge/module/x/gravity/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DrainModuleAccount sends all the funds from the module account to the shutdownDrainAddr
func (k Keeper) DrainModuleAccount(ctx sdk.Context) error {
	modAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName).GetAddress()
	balances := k.bankKeeper.GetAllBalances(ctx, modAcc)
	if balances.Empty() {
		return nil
	}
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, k.shutdownDrainAddr, balances)
}
