package chain

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"

	"github.com/ignite-hq/cli/ignite/pkg/cache"
	"github.com/ignite-hq/cli/ignite/pkg/checksum"
	"github.com/ignite-hq/cli/ignite/pkg/cmdrunner"
	"github.com/ignite-hq/cli/ignite/pkg/cmdrunner/exec"
	"github.com/ignite-hq/cli/ignite/pkg/cmdrunner/step"
	"github.com/ignite-hq/cli/ignite/pkg/dirchange"
	"github.com/ignite-hq/cli/ignite/pkg/goanalysis"
	"github.com/ignite-hq/cli/ignite/pkg/gocmd"
	"github.com/ignite-hq/cli/ignite/pkg/xstrings"
)

const (
	releaseDir                   = "release"
	releaseChecksumKey           = "release_checksum"
	modChecksumKey               = "go_mod_checksum"
	buildDirchangeCacheNamespace = "build.dirchange"
)

// Build builds and installs app binaries.
func (c *Chain) Build(ctx context.Context, cacheStorage cache.Storage, output string) (binaryName string, err error) {
	if err := c.setup(); err != nil {
		return "", err
	}

	if err := c.build(ctx, cacheStorage, output); err != nil {
		return "", err
	}

	return c.Binary()
}

func (c *Chain) build(ctx context.Context, cacheStorage cache.Storage, output string) (err error) {
	defer func() {
		var exitErr *exec.ExitError

		if errors.As(err, &exitErr) || errors.Is(err, goanalysis.ErrMultipleMainPackagesFound) {
			err = &CannotBuildAppError{err}
		}
	}()

	if err := c.generateAll(ctx, cacheStorage); err != nil {
		return err
	}

	buildFlags, err := c.preBuild(ctx, cacheStorage)
	if err != nil {
		return err
	}

	binary, err := c.Binary()
	if err != nil {
		return err
	}

	path, err := c.discoverMain(c.app.Path)
	if err != nil {
		return err
	}

	return gocmd.BuildPath(ctx, output, binary, path, buildFlags)
}

// BuildRelease builds binaries for a release. targets is a list
// of GOOS:GOARCH when provided. It defaults to your system when no targets provided.
// prefix is used as prefix to tarballs containing each target.
func (c *Chain) BuildRelease(ctx context.Context, cacheStorage cache.Storage, output, prefix string, targets ...string) (releasePath string, err error) {
	if prefix == "" {
		prefix = c.app.Name
	}
	if len(targets) == 0 {
		targets = []string{gocmd.BuildTarget(runtime.GOOS, runtime.GOARCH)}
	}

	// prepare for build.
	if err := c.setup(); err != nil {
		return "", err
	}

	buildFlags, err := c.preBuild(ctx, cacheStorage)
	if err != nil {
		return "", err
	}

	binary, err := c.Binary()
	if err != nil {
		return "", err
	}

	mainPath, err := c.discoverMain(c.app.Path)
	if err != nil {
		return "", err
	}

	releasePath = output
	if releasePath == "" {
		releasePath = filepath.Join(c.app.Path, releaseDir)
		// reset the release dir.
		if err := os.RemoveAll(releasePath); err != nil {
			return "", err
		}
	}

	if err := os.MkdirAll(releasePath, 0755); err != nil {
		return "", err
	}

	for _, t := range targets {
		// build binary for a target, tarball it and save it under the release dir.
		goos, goarch, err := gocmd.ParseTarget(t)
		if err != nil {
			return "", err
		}

		out, err := os.MkdirTemp("", "")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(out)

		buildOptions := []exec.Option{
			exec.StepOption(step.Env(
				cmdrunner.Env(gocmd.EnvGOOS, goos),
				cmdrunner.Env(gocmd.EnvGOARCH, goarch),
			)),
		}

		if err := gocmd.BuildPath(ctx, out, binary, mainPath, buildFlags, buildOptions...); err != nil {
			return "", err
		}

		tarr, err := archive.Tar(out, archive.Gzip)
		if err != nil {
			return "", err
		}

		tarName := fmt.Sprintf("%s_%s_%s.tar.gz", prefix, goos, goarch)
		tarPath := filepath.Join(releasePath, tarName)

		tarf, err := os.Create(tarPath)
		if err != nil {
			return "", err
		}
		defer tarf.Close()

		if _, err := io.Copy(tarf, tarr); err != nil {
			return "", err
		}
		tarf.Close()
	}

	checksumPath := filepath.Join(releasePath, releaseChecksumKey)

	// create a checksum.txt and return with the path to release dir.
	return releasePath, checksum.Sum(releasePath, checksumPath)
}

func (c *Chain) preBuild(ctx context.Context, cacheStorage cache.Storage) (buildFlags []string, err error) {
	config, err := c.Config()
	if err != nil {
		return nil, err
	}

	chainID, err := c.ID()
	if err != nil {
		return nil, err
	}

	ldFlags := config.GetBuild().LDFlags
	ldFlags = append(ldFlags,
		fmt.Sprintf("-X github.com/cosmos/cosmos-sdk/version.Name=%s", xstrings.Title(c.app.Name)),
		fmt.Sprintf("-X github.com/cosmos/cosmos-sdk/version.AppName=%sd", c.app.Name),
		fmt.Sprintf("-X github.com/cosmos/cosmos-sdk/version.Version=%s", c.sourceVersion.tag),
		fmt.Sprintf("-X github.com/cosmos/cosmos-sdk/version.Commit=%s", c.sourceVersion.hash),
		fmt.Sprintf("-X %s/cmd/%s/cmd.ChainID=%s", c.app.ImportPath, c.app.D(), chainID),
	)
	buildFlags = []string{
		gocmd.FlagMod, gocmd.FlagModValueReadOnly,
		gocmd.FlagLdflags, gocmd.Ldflags(ldFlags...),
	}

	fmt.Fprintln(c.stdLog().out, "📦 Installing dependencies...")

	// We do mod tidy before checking for checksum changes, because go.mod gets modified often
	// and the mod verify command is the expensive one anyway
	if err := gocmd.ModTidy(ctx, c.app.Path); err != nil {
		return nil, err
	}

	dirCache := cache.New[[]byte](cacheStorage, buildDirchangeCacheNamespace)
	modChanged, err := dirchange.HasDirChecksumChanged(dirCache, modChecksumKey, c.app.Path, "go.mod")
	if err != nil {
		return nil, err
	}

	if modChanged {
		if err := gocmd.ModVerify(ctx, c.app.Path); err != nil {
			return nil, err
		}

		if err := dirchange.SaveDirChecksum(dirCache, modChecksumKey, c.app.Path, "go.mod"); err != nil {
			return nil, err
		}
	}

	fmt.Fprintln(c.stdLog().out, "🛠️  Building the blockchain...")

	return buildFlags, nil
}

func (c *Chain) discoverMain(path string) (pkgPath string, err error) {
	conf, err := c.Config()
	if err != nil {
		return "", err
	}

	if conf.GetBuild().Main != "" {
		return filepath.Join(c.app.Path, conf.GetBuild().Main), nil
	}

	path, err = goanalysis.DiscoverOneMain(path)
	if err == goanalysis.ErrMultipleMainPackagesFound {
		return "", errors.Wrap(err, "specify the path to your chain's main package in your config.yml>build.main")
	}
	return path, err
}
