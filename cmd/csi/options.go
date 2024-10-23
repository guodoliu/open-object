package csi

import (
	"github.com/guodoliu/csi-driver-s3/pkg/common"
	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
)

type csiOption struct {
	Endpoint     string
	NodeID       string
	Driver       string
	Master       string
	KubeConfig   string
	FeatureGates map[string]bool
}

func (opt *csiOption) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&opt.Endpoint, "endpoint", common.DefaultEndpoint, "CSI endpoint")
	fs.StringVar(&opt.NodeID, "nodeID", "", "node id")
	fs.StringVar(&opt.Driver, "driver", common.DefaultDriverName, "csi driver name")
	fs.StringVar(&opt.Master, "master", "", "URL/IP for master")
	fs.StringVar(&opt.KubeConfig, "kubeconfig", "", "path to the kubeconfig file")
	fs.Var(cliflag.NewMapStringBool(&opt.FeatureGates), "feature-gates", "A set of key=value pairs that describe feature gates for alpha/experimental features.")
}
