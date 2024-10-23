package connector

import (
	"github.com/guodoliu/csi-driver-s3/pkg/common"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "connector",
	Short: "command for running fuse connector in host",
	Run: func(cmd *cobra.Command, args []string) {
		common.RunConnector()
	},
}
