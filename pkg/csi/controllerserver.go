package csi

import (
	"context"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/guodoliu/csi-driver-s3/pkg/common"
	csi_common "github.com/guodoliu/csi-driver-s3/pkg/csi/csi-common"
	"github.com/guodoliu/csi-driver-s3/pkg/csi/s3minio"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"net"
	"net/url"
	"strings"
)

type controllerServer struct {
	kubeClient *kubernetes.Clientset
	*csi_common.DefaultControllerServer
}

func newControllerServer(d *csi_common.CSIDriver) *controllerServer {
	cfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	return &controllerServer{
		DefaultControllerServer: csi_common.NewDefaultControllerServer(d),
		kubeClient:              kubeClient,
	}
}

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if len(req.GetName()) == 0 {
		return &csi.CreateVolumeResponse{}, status.Error(codes.InvalidArgument, "Name missing in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return &csi.CreateVolumeResponse{}, status.Error(codes.InvalidArgument, "Volume Capabilities missing in request")
	}

	// get driver
	var err error
	driverName := getDriverName(req.GetParameters())
	if len(driverName) == 0 {
		return &csi.CreateVolumeResponse{}, status.Errorf(codes.InvalidArgument, "%s not found in storageclass parameters", common.ParamDriverName)
	}
	var driver Driver
	switch driverName {
	case s3minio.DriverName:
		driver, err = getMinIODriver(req.Secrets)
		if err != nil {
			return &csi.CreateVolumeResponse{}, status.Errorf(codes.InvalidArgument, "fail to get minio driver: %s", err.Error())
		}
	default:
		return &csi.CreateVolumeResponse{}, status.Errorf(codes.Internal, "unknown storage driver: %s", driverName)
	}

	// create volume
	return driver.CreateVolume(ctx, req)
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return &csi.DeleteVolumeResponse{}, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	pv, err := cs.kubeClient.CoreV1().PersistentVolumes().Get(context.Background(), req.GetVolumeId(), metav1.GetOptions{})
	if err != nil {
		return &csi.DeleteVolumeResponse{}, status.Error(codes.Internal, err.Error())
	}

	// get driver
	driverName := getDriverName(pv.Spec.CSI.VolumeAttributes)
	if driverName == "" {
		return &csi.DeleteVolumeResponse{}, status.Errorf(codes.InvalidArgument, "%s not found in pv %s attributes", common.ParamDriverName, pv.Name)
	}
	var driver Driver
	switch driverName {
	case s3minio.DriverName:
		driver, err = getMinIODriver(req.Secrets)
		if err != nil {
			return &csi.DeleteVolumeResponse{}, status.Errorf(codes.Internal, "fail to get minio driver: %s", err.Error())
		}
	default:
		return &csi.DeleteVolumeResponse{}, status.Errorf(codes.Internal, "unknown driver: %s", driverName)
	}

	return driver.DeleteVolume(ctx, req)
}

func (cs *controllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return &csi.ControllerExpandVolumeResponse{}, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	pv, err := cs.kubeClient.CoreV1().PersistentVolumes().Get(context.Background(), req.GetVolumeId(), metav1.GetOptions{})
	if err != nil {
		return &csi.ControllerExpandVolumeResponse{}, fmt.Errorf("failed to get pv %s info: %s", req.GetVolumeId(), err.Error())
	}

	// get driver
	driverName := getDriverName(pv.Spec.CSI.VolumeAttributes)
	if driverName == "" {
		return &csi.ControllerExpandVolumeResponse{}, status.Errorf(codes.InvalidArgument, "%s not found in pv %s attributes", common.ParamDriverName, pv.Name)
	}
	var driver Driver
	switch driverName {
	case s3minio.DriverName:
		driver, err = getMinIODriver(req.Secrets)
		if err != nil {
			return &csi.ControllerExpandVolumeResponse{}, status.Errorf(codes.Internal, "fail to get minio driver: %s", err.Error())
		}
	default:
		return &csi.ControllerExpandVolumeResponse{}, status.Errorf(codes.Internal, "unknown driver: %s", driverName)
	}

	// expand volume
	return driver.ControllerExpandVolume(ctx, req)
}

// new add
func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return cs.DefaultControllerServer.ControllerPublishVolume(ctx, req)
}

func (cs *controllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return cs.DefaultControllerServer.ListSnapshots(ctx, req)
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return cs.DefaultControllerServer.ValidateVolumeCapabilities(ctx, req)
}

func (cs *controllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return cs.DefaultControllerServer.DeleteSnapshot(ctx, req)
}

func (cs *controllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return cs.DefaultControllerServer.ControllerGetCapabilities(ctx, req)
}

func (cs *controllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return cs.DefaultControllerServer.CreateSnapshot(ctx, req)
}

func (cs *controllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return cs.DefaultControllerServer.ListVolumes(ctx, req)
}

func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return cs.DefaultControllerServer.ControllerUnpublishVolume(ctx, req)
}

func (cs *controllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func getDriverName(attr map[string]string) string {
	return attr[common.ParamDriverName]
}

func getMinIODriver(secrets map[string]string) (Driver, error) {
	endpoint, err := GetS3Endpoint(secrets[s3minio.SecretMinIOHost])
	if err != nil {
		return nil, err
	}
	klog.Infof("endpoint: %s", endpoint)

	//cfg := &s3minio.S3Config{
	//	AK:       secrets[s3minio.SecretAK],
	//	SK:       secrets[s3minio.SecretSK],
	//	Region:   secrets[s3minio.SecretRegion],
	//	Endpoint: endpoint,
	//}
	cfg := &s3minio.S3Config{
		AK:       "NvmbqoUxn50jqcHlHBEG",
		SK:       "EAGogws8UcIA8WuBMOuGDRKIC7MXvWcswD3dHpzW",
		Region:   "china",
		Endpoint: "http://192.168.165.50:9000",
	}
	return s3minio.NewMinIODriver(cfg)
}

func GetS3Endpoint(s3host string) (string, error) {
	// endpoint
	endpoint := ""
	u, err := url.Parse(s3host)
	if err != nil {
		return "", err
	}
	scheme := strings.ToLower(u.Scheme)
	host := u.Hostname()
	port := u.Port()
	// check if is ip
	addr := net.ParseIP(host)
	if addr == nil {
		// is not ip
		IPs, err := net.LookupIP(host)
		if err != nil {
			return "", fmt.Errorf("fail to lookup %s: %s", host, err.Error())
		}

		if len(IPs) > 0 {
			addr = IPs[0]
		} else {
			return "", fmt.Errorf("no ip found when lookup host %s", host)
		}
	}

	if addr.To4() == nil {
		endpoint = fmt.Sprintf("%s://[%s]:%s", scheme, addr.String(), port)
	} else {
		endpoint = fmt.Sprintf("%s://%s:%s", scheme, addr.String(), port)
	}

	return endpoint, nil
}
