package ignitecmd

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/rdegges/go-ipify"
	"github.com/spf13/cobra"

	"github.com/ignite-hq/cli/ignite/pkg/cliquiz"
	"github.com/ignite-hq/cli/ignite/pkg/clispinner"
	"github.com/ignite-hq/cli/ignite/pkg/gitpod"
	"github.com/ignite-hq/cli/ignite/pkg/xchisel"
	"github.com/ignite-hq/cli/ignite/services/network"
	"github.com/ignite-hq/cli/ignite/services/network/networkchain"
)

const (
	defaultPort     = "26656"
	flagGentx       = "gentx"
	flagAmount      = "amount"
	flagDefaultPeer = "default-peer"
)

// NewNetworkChainJoin creates a new chain join command to join
// to a network as a validator.
func NewNetworkChainJoin() *cobra.Command {
	c := &cobra.Command{
		Use:   "join [launch-id]",
		Short: "Request to join a network as a validator",
		Args:  cobra.ExactArgs(1),
		RunE:  networkChainJoinHandler,
	}
	c.Flags().String(flagGentx, "", "Path to a gentx json file")
	c.Flags().String(flagAmount, "", "Amount of coins for account request")
	c.Flags().Bool(flagDefaultPeer, false, "Use default ip address as peer")
	c.Flags().AddFlagSet(flagNetworkFrom())
	c.Flags().AddFlagSet(flagSetHome())
	c.Flags().AddFlagSet(flagSetKeyringBackend())
	c.Flags().AddFlagSet(flagSetYes())
	return c
}

func networkChainJoinHandler(cmd *cobra.Command, args []string) error {
	var (
		gentxPath, _   = cmd.Flags().GetString(flagGentx)
		amount, _      = cmd.Flags().GetString(flagAmount)
		defaultPeer, _ = cmd.Flags().GetBool(flagDefaultPeer)
	)

	nb, err := newNetworkBuilder(cmd)
	if err != nil {
		return err
	}
	defer nb.Cleanup()

	// parse launch ID.
	launchID, err := network.ParseID(args[0])
	if err != nil {
		return err
	}

	joinOptions := []network.JoinOption{
		network.WithCustomGentxPath(gentxPath),
	}

	// if there is no custom gentx, we need to detect the public address.
	if gentxPath == "" {
		// get the peer public address for the validator.
		var publicAddr string
		if !defaultPeer {
			publicAddr, err = askPublicAddress(cmd.Context(), nb.Spinner)
		} else {
			publicAddr, err = ipify.GetIp()
			publicAddr = fmt.Sprintf("%s:%s", publicAddr, defaultPort)
		}
		if err != nil {
			return err
		}

		joinOptions = append(joinOptions, network.WithPublicAddress(publicAddr))
	}

	n, err := nb.Network()
	if err != nil {
		return err
	}

	chainLaunch, err := n.ChainLaunch(cmd.Context(), launchID)
	if err != nil {
		return err
	}

	c, err := nb.Chain(networkchain.SourceLaunch(chainLaunch))
	if err != nil {
		return err
	}

	if amount != "" {
		// parse the amount.
		amountCoins, err := sdk.ParseCoinsNormalized(amount)
		if err != nil {
			return errors.Wrap(err, "error parsing amount")
		}
		joinOptions = append(joinOptions, network.WithAccountRequest(amountCoins))
	} else {
		nb.Spinner.Stop()

		if !getYes(cmd) {
			label := fmt.Sprintf("You haven't set the --%s flag and therefore an account request won't be submitted. Do you confirm", flagAmount)
			prompt := promptui.Prompt{
				Label:     label,
				IsConfirm: true,
			}
			if _, err := prompt.Run(); err != nil {
				fmt.Println("said no")
				return nil
			}
		}

		fmt.Printf("%s %s\n", clispinner.Info, "Account request won't be submitted")
		nb.Spinner.Start()
	}

	// create the message to add the validator.
	return n.Join(cmd.Context(), c, launchID, joinOptions...)
}

// askPublicAddress prepare questions to interactively ask for a publicAddress
// when peer isn't provided and not running through chisel proxy.
func askPublicAddress(ctx context.Context, s *clispinner.Spinner) (publicAddress string, err error) {
	s.Stop()
	defer s.Start()

	options := []cliquiz.Option{
		cliquiz.Required(),
	}
	if gitpod.IsOnGitpod() {
		publicAddress, err = gitpod.URLForPort(ctx, xchisel.DefaultServerPort)
		if err != nil {
			return "", errors.Wrap(err, "cannot read public Gitpod address of the node")
		}
		return publicAddress, nil
	}

	// even if GetIp fails we won't handle the error because we don't want to interrupt a join process.
	// just in case if GetIp fails user should enter his address manually
	ip, err := ipify.GetIp()
	if err == nil {
		options = append(options, cliquiz.DefaultAnswer(fmt.Sprintf("%s:%s", ip, defaultPort)))
	}

	questions := []cliquiz.Question{cliquiz.NewQuestion(
		"Peer's address",
		&publicAddress,
		options...,
	)}
	return publicAddress, cliquiz.Ask(questions...)
}
