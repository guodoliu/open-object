package csi

import (
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/guodoliu/csi-driver-s3/pkg/common"
	csi_common "github.com/guodoliu/csi-driver-s3/pkg/csi/csi-common"
	"github.com/guodoliu/csi-driver-s3/pkg/csi/s3minio"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

type nodeServer struct {
	*csi_common.DefaultNodeServer
}

func newNodeServer(d *csi_common.CSIDriver) *nodeServer {
	return &nodeServer{
		DefaultNodeServer: csi_common.NewDefaultNodeServer(d),
	}
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()

	// check arguments
	if req.GetVolumeCapability() == nil {
		return &csi.NodePublishVolumeResponse{}, status.Error(codes.InvalidArgument, "Volume capability missing in request")
	}
	if len(volumeID) == 0 {
		return &csi.NodePublishVolumeResponse{}, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(targetPath) == 0 {
		return &csi.NodePublishVolumeResponse{}, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	// get driver
	var err error
	driverName := getDriverName(req.GetVolumeContext())
	if driverName == "" {
		return &csi.NodePublishVolumeResponse{}, status.Errorf(codes.InvalidArgument, "%s not found in storageclass parameters", common.ParamDriverName)
	}
	var driver Driver
	switch driverName {
	case s3minio.DriverName:
		driver, err = getMinIODriver(req.Secrets)
		if err != nil {
			return &csi.NodePublishVolumeResponse{}, status.Errorf(codes.Internal, "fail to get minio driver: %s", err.Error())
		}
	default:
		return &csi.NodePublishVolumeResponse{}, status.Errorf(codes.Internal, "unknown driver: %s", driverName)
	}

	return driver.NodePublishVolume(ctx, req)
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()

	// Check arguments
	if len(volumeID) == 0 {
		return &csi.NodeUnpublishVolumeResponse{}, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(targetPath) == 0 {
		return &csi.NodeUnpublishVolumeResponse{}, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	if err := common.FuseUmount(targetPath); err != nil {
		return &csi.NodeUnpublishVolumeResponse{}, status.Error(codes.Internal, err.Error())
	}
	klog.Infof("s3: mountpoint %s has been unmounted.", targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return &csi.NodeExpandVolumeResponse{}, status.Error(codes.Unimplemented, "NodeExpandVolume is not implemented")
}

func (ns *nodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, nil
}

func (ns *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return ns.DefaultNodeServer.NodeGetInfo(ctx, req)
}

func (ns *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	nscap := &csi.NodeServiceCapability{
		Type: &csi.NodeServiceCapability_Rpc{
			Rpc: &csi.NodeServiceCapability_RPC{
				Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
			},
		},
	}

	nscap3 := &csi.NodeServiceCapability{
		Type: &csi.NodeServiceCapability_Rpc{
			Rpc: &csi.NodeServiceCapability_RPC{
				Type: csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
			},
		},
	}

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			nscap,
			nscap3,
		},
	}, nil
}
