package csi

import (
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi"
	csi_common "github.com/guodoliu/csi-driver-s3/pkg/csi/csi-common"
	"github.com/guodoliu/csi-driver-s3/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type Driver interface {
	CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error)
	DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error)
	ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error)
	NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error)
	NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error)
	NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error)
	NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error)
	NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error)
	NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error)
}

type FuseDriver struct {
	driver   *csi_common.CSIDriver
	endpoint string

	ids *identityServer
	ns  *nodeServer
	cs  *controllerServer
}

func NewFuseDriver(nodeID, endpoint, driverName string, kubeClient *kubernetes.Clientset) (*FuseDriver, error) {
	driver := csi_common.NewCSIDriver(driverName, version.Version, nodeID)
	if driver == nil {
		klog.Fatalln("Failed to initialize CSI Driver.")
	}

	s3Driver := &FuseDriver{
		endpoint: endpoint,
		driver:   driver,
		ids:      newIdentityServer(driver),
		cs:       newControllerServer(driver),
		ns:       newNodeServer(driver),
	}
	return s3Driver, nil
}

func (s3 *FuseDriver) Run() {
	// Initialize default library driver
	s3.driver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
	})
	s3.driver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	})
	s := csi_common.NewNonBlockingGRPCServer()
	s.Start(s3.endpoint, s3.ids, s3.cs, s3.ns)
	s.Wait()
}
