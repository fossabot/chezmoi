package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	vfs "github.com/twpayne/go-vfs"
)

type dataCmdConfig struct {
	format string
}

var dataCmd = &cobra.Command{
	Use:     "data",
	Args:    cobra.NoArgs,
	Short:   "Write the template data to stdout",
	Long:    mustGetLongHelp("data"),
	Example: getExample("data"),
	PreRunE: config.ensureNoError,
	RunE:    makeRunE(config.runDataCmd),
}

func init() {
	rootCmd.AddCommand(dataCmd)

	persistentFlags := dataCmd.PersistentFlags()
	persistentFlags.StringVarP(&config.data.format, "format", "f", "json", "format (JSON, TOML, or YAML)")
}

func (c *Config) runDataCmd(fs vfs.FS, args []string) error {
	format, ok := formatMap[strings.ToLower(c.data.format)]
	if !ok {
		return fmt.Errorf("%s: unknown format", c.data.format)
	}
	ts, err := c.getTargetState(fs, nil)
	if err != nil {
		return err
	}
	return format(c.Stdout(), ts.Data)
}
