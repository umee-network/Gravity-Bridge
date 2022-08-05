package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	crypto "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmtypes "github.com/tendermint/tendermint/types"

	gravitytypes "github.com/Gravity-Bridge/Gravity-Bridge/module/x/gravity/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/staking/client/cli"
)

// GenTxCmd builds the application's gentx command.
//nolint:gocyclo
func GenTxCmd(mbm module.BasicManager, txEncCfg client.TxEncodingConfig, genBalIterator types.GenesisBalancesIterator, defaultNodeHome string) *cobra.Command {
	ipDefault, errIpDefault := server.ExternalIP()
	if errIpDefault != nil {
		fmt.Printf("errIpDefault %v", errIpDefault)
	}
	fsCreateValidator, defaultsDesc := cli.CreateValidatorMsgFlagSet(ipDefault)

	//nolint: exhaustivestruct
	cmd := &cobra.Command{
		Use:   "gentx [key_name] [amount] [eth-address] [orchestrator-address]",
		Short: "Generate a genesis tx carrying a self delegation, oracle key delegation and orchestrator key delegation",
		Args:  cobra.ExactArgs(4),
		Long: fmt.Sprintf(`Generate a genesis transaction that creates a validator with a self-delegation, oracle key 
delegation and orchestrator key delegation that is signed by the key in the Keyring referenced by a given name. A node 
ID and Bech32 consensus pubkey may optionally be provided. If they are omitted, they will be retrieved from the 
priv_validator.json file. The following default parameters are included:
    %s

Example:
$ %s gentx my-key-name 1000000stake 0x033030FEeBd93E3178487c35A9c8cA80874353C9 cosmos1ahx7f8wyertuus9r20284ej0asrs085case3kn --home=/path/to/home/dir --keyring-backend=os --chain-id=test-chain-1 \
    --moniker="myvalidator" \
    --commission-max-change-rate=0.01 \
    --commission-max-rate=1.0 \
    --commission-rate=0.07 \
    --details="..." \
    --security-contact="..." \
    --website="..."
`, defaultsDesc, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			cdc := clientCtx.Codec

			config := serverCtx.Config
			homeFlag, errHomeFlag := cmd.Flags().GetString(flags.FlagHome)
			if errHomeFlag != nil {
				fmt.Printf("errHomeFlag %v", errHomeFlag)
			}

			if len(homeFlag) > 0 {
				config = config.SetRoot(homeFlag)
			} else {
				config = config.SetRoot(clientCtx.HomeDir)
			}
			nodeID, valPubKey, err := genutil.InitializeNodeValidatorFiles(serverCtx.Config)
			if err != nil {
				return errors.Wrap(err, "failed to initialize node validator files")
			}

			// read --nodeID, if empty take it from priv_validator.json
			if nodeIDString, errNodeIDString := cmd.Flags().GetString(cli.FlagNodeID); nodeIDString != "" {
				if errNodeIDString != nil {
					fmt.Printf("errNodeIDString %v", errNodeIDString)
				}
				nodeID = nodeIDString
			}

			// read --pubkey, if empty take it from priv_validator.json
			if valPubKeyString, errValPubKeyString := cmd.Flags().GetString(cli.FlagPubKey); valPubKeyString != "" {
				if errValPubKeyString != nil {
					fmt.Printf("errValPubKeyString %v", errValPubKeyString)
				}
				var valPubKey crypto.PubKey
				err := clientCtx.Codec.UnmarshalInterfaceJSON([]byte(valPubKeyString), &valPubKey)
				if err != nil {
					return errors.Wrap(err, "failed to get consensus node public key")
				}
			}

			genDoc, err := tmtypes.GenesisDocFromFile(config.GenesisFile())
			if err != nil {
				return errors.Wrapf(err, "failed to read genesis doc file %s", config.GenesisFile())
			}

			var genesisState map[string]json.RawMessage
			if err = json.Unmarshal(genDoc.AppState, &genesisState); err != nil {
				return errors.Wrap(err, "failed to unmarshal genesis state")
			}

			if err = mbm.ValidateGenesis(cdc, txEncCfg, genesisState); err != nil {
				return errors.Wrap(err, "failed to validate genesis state")
			}

			inBuf := bufio.NewReader(cmd.InOrStdin())

			name := args[0]
			key, err := clientCtx.Keyring.Key(name)
			if err != nil {
				return errors.Wrapf(err, "failed to fetch '%s' from the keyring", name)
			}

			ethAddress := args[2]

			if err := gravitytypes.ValidateEthAddress(ethAddress); err != nil {
				return errors.Wrapf(err, "invalid ethereum address")
			}

			orchAddress, err := sdk.AccAddressFromBech32(args[3])
			if err != nil {
				return errors.Wrapf(err, "failed to parse orchAddress(%s)", args[3])
			}

			moniker := config.Moniker
			if m, errFlagGetString := cmd.Flags().GetString(cli.FlagMoniker); m != "" {
				if errFlagGetString != nil {
					fmt.Printf("FlagGetString has an error")
				}
				moniker = m
			}

			// set flags for creating a gentx
			createValCfg, err := cli.PrepareConfigForTxCreateValidator(cmd.Flags(), moniker, nodeID, genDoc.ChainID, valPubKey)
			if err != nil {
				return errors.Wrap(err, "error creating configuration to create validator msg")
			}

			amount := args[1]
			coins, err := sdk.ParseCoinsNormalized(amount)
			if err != nil {
				return errors.Wrap(err, "failed to parse coins")
			}

			// validate validator account in genesis
			addr, err := key.GetAddress()
			if err != nil {
				return errors.Wrap(err, "failed to get validator address")
			}
			if err = genutil.ValidateAccountInGenesis(genesisState, genBalIterator, addr, coins, cdc); err != nil {
				return errors.Wrap(err, "failed to validate validator account in genesis")
			}

			txFactory := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return errors.Wrap(err, "error creating tx builder")
			}

			clientCtx = clientCtx.WithInput(inBuf).WithFromAddress(addr)

			// The following line comes from a discrepancy between the `gentx`
			// and `create-validator` commands:
			// - `gentx` expects amount as an arg,
			// - `create-validator` expects amount as a required flag.
			// ref: https://github.com/cosmos/cosmos-sdk/issues/8251
			// Since gentx doesn't set the amount flag (which `create-validator`
			// reads from), we copy the amount arg into the valCfg directly.
			//
			// Ideally, the `create-validator` command should take a validator
			// config file instead of so many flags.
			// ref: https://github.com/cosmos/cosmos-sdk/issues/8177
			createValCfg.Amount = amount

			// create a 'create-validator' message
			txBldr, msg, err := cli.BuildCreateValidatorMsg(clientCtx, createValCfg, txFactory, true)
			if err != nil {
				return errors.Wrap(err, "failed to build create-validator message")
			}

			delegateKeySetMsg := &gravitytypes.MsgSetOrchestratorAddress{
				Validator:    sdk.ValAddress(addr).String(),
				Orchestrator: orchAddress.String(),
				EthAddress:   ethAddress,
			}

			msgs := []sdk.Msg{msg, delegateKeySetMsg}

			if key.GetType() == keyring.TypeOffline || key.GetType() == keyring.TypeMulti {
				cmd.PrintErrln("Offline key passed in. Use `tx sign` command to sign.")
				return txBldr.PrintUnsignedTx(clientCtx, msgs...)
			}

			// write the unsigned transaction to the buffer
			w := bytes.NewBuffer([]byte{})
			clientCtx = clientCtx.WithOutput(w)

			if err = txBldr.PrintUnsignedTx(clientCtx, msgs...); err != nil {
				return errors.Wrap(err, "failed to print unsigned std tx")
			}

			// read the transaction
			stdTx, err := readUnsignedGenTxFile(clientCtx, w)
			if err != nil {
				return errors.Wrap(err, "failed to read unsigned gen tx file")
			}

			// sign the transaction and write it to the output file
			txBuilder, err := clientCtx.TxConfig.WrapTxBuilder(stdTx)
			if err != nil {
				return fmt.Errorf("error creating tx builder: %w", err)
			}

			err = authclient.SignTx(txFactory, clientCtx, name, txBuilder, true, true)
			if err != nil {
				return errors.Wrap(err, "failed to sign std tx")
			}

			outputDocument, errOutputDocument := cmd.Flags().GetString(flags.FlagOutputDocument)
			if errOutputDocument != nil {
				fmt.Printf("OutputDocument has an error")
			}
			if outputDocument == "" {
				outputDocument, err = makeOutputFilepath(config.RootDir, nodeID)
				if err != nil {
					return errors.Wrap(err, "failed to create output file path")
				}
			}

			if err := writeSignedGenTx(clientCtx, outputDocument, stdTx); err != nil {
				return errors.Wrap(err, "failed to write signed gen tx")
			}

			cmd.PrintErrf("Genesis transaction written to %q\n", outputDocument)
			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	cmd.Flags().String(flags.FlagOutputDocument, "", "Write the genesis transaction JSON document to the given file instead of the default location")
	cmd.Flags().String(flags.FlagChainID, "", "The network chain ID")
	cmd.Flags().AddFlagSet(fsCreateValidator)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func makeOutputFilepath(rootDir, nodeID string) (string, error) {
	writePath := filepath.Join(rootDir, "config", "gentx")
	if err := tmos.EnsureDir(writePath, 0700); err != nil {
		return "", err
	}

	return filepath.Join(writePath, fmt.Sprintf("gentx-%v.json", nodeID)), nil
}

func readUnsignedGenTxFile(clientCtx client.Context, r io.Reader) (sdk.Tx, error) {
	bz, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	aTx, err := clientCtx.TxConfig.TxJSONDecoder()(bz)
	if err != nil {
		return nil, err
	}

	return aTx, err
}

func writeSignedGenTx(clientCtx client.Context, outputDocument string, tx sdk.Tx) error {
	outputFile, err := os.OpenFile(outputDocument, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	json, err := clientCtx.TxConfig.TxJSONEncoder()(tx)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(outputFile, "%s\n", json)

	return err
}

const flagGenTxDir = "gentx-dir"
