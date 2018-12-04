package cmd

// FIXME add --apply flag
// FIXME add --diff flag
// FIXME add --prompt flag
// FIXME add --recursive flag

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/twpayne/chezmoi/lib/chezmoi"
	vfs "github.com/twpayne/go-vfs"
)

var chattrCommand = &cobra.Command{
	Use:   "chattr",
	Args:  cobra.MinimumNArgs(2),
	Short: "Change the private, empty, executable, or template attributes of a target",
	RunE:  makeRunE(config.runChattrCommand),
}

type boolModifier int

type attributeModifiers struct {
	empty      boolModifier
	executable boolModifier
	private    boolModifier
	template   boolModifier
}

func init() {
	rootCommand.AddCommand(chattrCommand)
}

func (c *Config) runChattrCommand(fs vfs.FS, cmd *cobra.Command, args []string) error {
	ams, err := parseAttributeModifiers(args[0])
	if err != nil {
		return err
	}
	targetState, err := c.getTargetState(fs)
	if err != nil {
		return err
	}
	entries, err := c.getEntries(targetState, args[1:])
	if err != nil {
		return err
	}
	renames := make(map[string]string)
	for _, entry := range entries {
		oldSourceName := entry.SourceName()
		var newSourceName string
		switch entry := entry.(type) {
		case *chezmoi.Dir:
			psdn := chezmoi.ParseSourceDirName(oldSourceName)
			if private := ams.private.modify(entry.Private()); private {
				psdn.Perm &= 0700
			}
			newSourceName = psdn.SourceDirName()
		case *chezmoi.File:
			psfn := chezmoi.ParseSourceFileName(oldSourceName)
			mode := os.FileMode(0666)
			if executable := ams.executable.modify(entry.Executable()); executable {
				mode |= 0111
			}
			if private := ams.private.modify(entry.Private()); private {
				mode &= 0700
			}
			psfn.Mode = mode
			psfn.Empty = ams.empty.modify(entry.Empty)
			psfn.Template = ams.template.modify(entry.Template)
			newSourceName = psfn.SourceFileName()
		case *chezmoi.Symlink:
			psfn := chezmoi.ParseSourceFileName(oldSourceName)
			psfn.Template = ams.template.modify(entry.Template)
			newSourceName = psfn.SourceFileName()
		}
		if newSourceName != oldSourceName {
			renames[filepath.Join(targetState.SourceDir, oldSourceName)] = filepath.Join(targetState.SourceDir, newSourceName)
		}
	}

	actuator := c.getDefaultActuator(fs)

	// Sort oldpaths in reverse so we rename files before their parent
	// directories.
	var oldpaths []string
	for oldpath := range renames {
		oldpaths = append(oldpaths, oldpath)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(oldpaths)))
	for _, oldpath := range oldpaths {
		if err := actuator.Rename(oldpath, renames[oldpath]); err != nil {
			return err
		}
	}
	return nil
}

func parseAttributeModifiers(s string) (*attributeModifiers, error) {
	ams := &attributeModifiers{}
	for _, attributeModifier := range strings.Split(s, ",") {
		attributeModifier = strings.TrimSpace(attributeModifier)
		if attributeModifier == "" {
			continue
		}
		modifier := boolModifier(1)
		if attributeModifier[0] == '-' {
			modifier = boolModifier(-1)
		}
		attribute := attributeModifier
		if attributeModifier[0] == '-' || attributeModifier[0] == '+' {
			attribute = attributeModifier[1:]
		}
		switch attribute {
		case "empty", "e":
			ams.empty = modifier
		case "executable", "x":
			ams.executable = modifier
		case "private", "p":
			ams.private = modifier
		case "template", "t":
			ams.template = modifier
		default:
			return nil, fmt.Errorf("unknown attribute: %s", attribute)
		}
	}
	return ams, nil
}

func (bm boolModifier) modify(x bool) bool {
	switch {
	case bm < 0:
		return false
	case bm > 0:
		return true
	default:
		return x
	}
}
