package csi

import (
	"fmt"
	"github.com/guodoliu/csi-driver-s3/pkg/csi"
	"github.com/guodoliu/csi-driver-s3/pkg/csi/s3minio"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"os"
)

var (
	opt = csiOption{}
)

var Cmd = &cobra.Command{
	Use:   "csi",
	Short: "command for running csi plugin",
	Run: func(cmd *cobra.Command, args []string) {
		err := Start(&opt)
		if err != nil {
			klog.Fatalf("error: %s, quitting now\n", err.Error())
		}
	},
}

func init() {
	opt.addFlags(Cmd.Flags())
}

func Start(opt *csiOption) error {
	klog.Info("CSI Driver Name: %s, nodeID: %s, endPoints %s", opt.Driver, opt.NodeID, opt.Endpoint)

	if err := s3minio.DefaultMutableFeatureGate.SetFromMap(opt.FeatureGates); err != nil {
		return fmt.Errorf("unable to setup feature gates: %s", err.Error())
	}
	cfg, err := clientcmd.BuildConfigFromFlags(opt.Master, opt.KubeConfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	driver, err := csi.NewFuseDriver(opt.NodeID, opt.Endpoint, opt.Driver, kubeClient)
	if err != nil {
		klog.Fatal(err)
	}
	driver.Run()
	os.Exit(0)

	return nil
}
