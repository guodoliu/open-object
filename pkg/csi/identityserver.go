package csi

import (
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi"
	csi_common "github.com/guodoliu/csi-driver-s3/pkg/csi/csi-common"
)

type identityServer struct {
	*csi.UnimplementedIdentityServer
	*csi_common.DefaultIdentityServer
}

func newIdentityServer(d *csi_common.CSIDriver) *identityServer {
	return &identityServer{
		DefaultIdentityServer: csi_common.NewDefaultIdentityServer(d),
	}
}

func (ids *identityServer) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
			{
				Type: &csi.PluginCapability_VolumeExpansion_{
					VolumeExpansion: &csi.PluginCapability_VolumeExpansion{
						Type: csi.PluginCapability_VolumeExpansion_ONLINE,
					},
				},
			},
		},
	}, nil
}

func (ids *identityServer) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return ids.DefaultIdentityServer.GetPluginInfo(ctx, req)
}

func (ids *identityServer) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return ids.DefaultIdentityServer.Probe(ctx, req)
}

func (ids *identityServer) mustEmbedUnimplementedIdentityServer() {}
