package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/donaldgifford/claudelint/internal/config"
)

// newInitCmd returns the `init` subcommand, which writes a default
// .claudelint.hcl to the current directory. It refuses to clobber an
// existing file unless --force is passed so users do not accidentally
// lose their configuration.
func newInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold a default .claudelint.hcl in the current directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("resolve cwd: %w", err)
			}
			target := filepath.Join(cwd, config.Filename)

			if _, err := os.Stat(target); err == nil && !force {
				return fmt.Errorf(
					"%s already exists; re-run with --force to overwrite",
					config.Filename,
				)
			} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("stat %s: %w", config.Filename, err)
			}

			if err := os.WriteFile(target, []byte(config.DefaultScaffold), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", target, err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", config.Filename)
			return err
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing config file")
	return cmd
}
