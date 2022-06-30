package relayer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"golang.org/x/sync/errgroup"

	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	"github.com/ignite/cli/ignite/pkg/ctxticker"
	tsrelayer "github.com/ignite/cli/ignite/pkg/nodetime/programs/ts-relayer"
	relayerconf "github.com/ignite/cli/ignite/pkg/relayer/config"
	"github.com/ignite/cli/ignite/pkg/xurl"
)

const (
	ibcSetupGas   int64 = 2256000
	relayDuration       = time.Second * 5
)

// Relayer is an IBC relayer.
type Relayer struct {
	ca cosmosaccount.Registry
}

// New creates a new IBC relayer and uses ca to access accounts.
func New(ca cosmosaccount.Registry) Relayer {
	return Relayer{
		ca: ca,
	}
}

// Link links all chains that has a path to each other.
// paths are optional and acts as a filter to only link some chains.
// calling Link multiple times for the same paths does not have any side effects.
func (r Relayer) Link(ctx context.Context, pathIDs ...string) error {
	conf, err := relayerconf.Get()
	if err != nil {
		return err
	}

	for _, id := range pathIDs {
		path, err := conf.PathByID(id)
		if err != nil {
			return err
		}

		if path.Src.ChannelID != "" { // already linked.
			continue
		}

		if path, err = r.call(ctx, conf, path, "link"); err != nil {
			return err
		}

		if err := conf.UpdatePath(path); err != nil {
			return err
		}
		if err := relayerconf.Save(conf); err != nil {
			return err
		}
	}

	return nil
}

// Start relays packets for linked paths until ctx is canceled.
func (r Relayer) Start(ctx context.Context, pathIDs ...string) error {
	conf, err := relayerconf.Get()
	if err != nil {
		return err
	}

	wg, ctx := errgroup.WithContext(ctx)
	var m sync.Mutex // protects relayerconf.Path.

	start := func(id string) error {
		path, err := conf.PathByID(id)
		if err != nil {
			return err
		}

		if path, err = r.call(ctx, conf, path, "start"); err != nil {
			return err
		}

		m.Lock()
		defer m.Unlock()

		conf, err := relayerconf.Get()
		if err != nil {
			return err
		}

		if err := conf.UpdatePath(path); err != nil {
			return err
		}

		return relayerconf.Save(conf)
	}

	for _, id := range pathIDs {
		id := id

		wg.Go(func() error {
			return ctxticker.DoNow(ctx, relayDuration, func() error { return start(id) })
		})
	}

	return wg.Wait()
}

func (r Relayer) call(ctx context.Context, conf relayerconf.Config, path relayerconf.Path, action string) (
	reply relayerconf.Path, err error) {
	srcChain, srcKey, err := r.prepare(ctx, conf, path.Src.ChainID)
	if err != nil {
		return relayerconf.Path{}, err
	}

	dstChain, dstKey, err := r.prepare(ctx, conf, path.Dst.ChainID)
	if err != nil {
		return relayerconf.Path{}, err
	}

	args := []interface{}{
		path,
		srcChain,
		dstChain,
		srcKey,
		dstKey,
	}
	return reply, tsrelayer.Call(ctx, action, args, &reply)
}

func (r Relayer) prepare(ctx context.Context, conf relayerconf.Config, chainID string) (
	chain relayerconf.Chain, privKey string, err error) {
	chain, err = conf.ChainByID(chainID)
	if err != nil {
		return relayerconf.Chain{}, "", err
	}

	coins, err := r.balance(ctx, chain.RPCAddress, chain.Account, chain.AddressPrefix)
	if err != nil {
		return relayerconf.Chain{}, "", err
	}

	gasPrice, err := sdk.ParseCoinNormalized(chain.GasPrice)
	if err != nil {
		return relayerconf.Chain{}, "", err
	}

	account, err := r.ca.GetByName(chain.Account)
	if err != nil {
		return relayerconf.Chain{}, "", err
	}

	errMissingBalance := fmt.Errorf(`account "%s(%s)" on %q chain does not have enough balances`,
		account.Address(chain.AddressPrefix),
		chain.Account,
		chain.ID,
	)

	if len(coins) == 0 {
		return relayerconf.Chain{}, "", errMissingBalance
	}

	for _, coin := range coins {
		if gasPrice.Denom != coin.Denom {
			continue
		}

		if gasPrice.Amount.Int64()*ibcSetupGas > coin.Amount.Int64() {
			return relayerconf.Chain{}, "", errMissingBalance
		}
	}

	key, err := r.ca.ExportHex(chain.Account, "")
	if err != nil {
		return relayerconf.Chain{}, "", err
	}

	return chain, key, nil
}

func (r Relayer) balance(ctx context.Context, rpcAddress, account, addressPrefix string) (sdk.Coins, error) {
	client, err := cosmosclient.New(ctx, cosmosclient.WithNodeAddress(rpcAddress))
	if err != nil {
		return nil, err
	}

	acc, err := r.ca.GetByName(account)
	if err != nil {
		return nil, err
	}

	addr := acc.Address(addressPrefix)

	queryClient := banktypes.NewQueryClient(client.Context())
	res, err := queryClient.AllBalances(ctx, &banktypes.QueryAllBalancesRequest{Address: addr})
	if err != nil {
		return nil, err
	}

	return res.Balances, nil
}

// GetPath returns a path by its id.
func (r Relayer) GetPath(_ context.Context, id string) (relayerconf.Path, error) {
	conf, err := relayerconf.Get()
	if err != nil {
		return relayerconf.Path{}, err
	}

	return conf.PathByID(id)
}

// ListPaths list all the paths.
func (r Relayer) ListPaths(_ context.Context) ([]relayerconf.Path, error) {
	conf, err := relayerconf.Get()
	if err != nil {
		return nil, err
	}

	return conf.Paths, nil
}

func fixRPCAddress(rpcAddress string) string {
	return strings.TrimSuffix(xurl.HTTPEnsurePort(rpcAddress), "/")
}
