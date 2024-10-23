package version

import (
	"fmt"
	"github.com/guodoliu/csi-driver-s3/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	verbose bool
)

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of open-local",
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			fmt.Printf("%s\n", version.NameWithVersion(true))
		} else {
			fmt.Printf("%s\n", version.NameWithVersion(false))
		}
	},
}

func init() {
	addFlags(Cmd.Flags())
}

func addFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&verbose, "verbose", false, "show detailed version info")
}
