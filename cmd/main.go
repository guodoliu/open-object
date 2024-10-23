package main

import (
	"flag"
	"fmt"
	"github.com/guodoliu/csi-driver-s3/cmd/connector"
	"github.com/guodoliu/csi-driver-s3/cmd/csi"
	"github.com/guodoliu/csi-driver-s3/cmd/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
	"os"
	"strings"
)

var (
	mainCmd = &cobra.Command{
		Use: "open-object",
	}
	VERSION  = ""
	COMMITID = ""
)

func init() {
	flag.Parse()
	mainCmd.SetGlobalNormalizationFunc(wordSepNormalizeFunc)
	mainCmd.DisableAutoGenTag = true
	addCommands()
}

func main() {
	klog.Info("Version: %s, Commit: %s", VERSION, COMMITID)
	if err := mainCmd.Execute(); err != nil {
		fmt.Printf("open-object start error: %+v\n", err)
		os.Exit(1)
	}
}

func addCommands() {
	mainCmd.AddCommand(
		csi.Cmd,
		version.Cmd,
		connector.Cmd)
}

// wordSepNormalizeFunc changes all flags that contain "_" separators
func wordSepNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	if strings.Contains(name, "_") {
		return pflag.NormalizedName(strings.Replace(name, "_", "-", -1))
	}
	return pflag.NormalizedName(name)
}
