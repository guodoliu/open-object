package s3minio

import (
	"context"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/guodoliu/csi-driver-s3/pkg/common"
	"github.com/minio/madmin-go/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"k8s.io/utils/mount"
	"os"
	"strconv"
	"strings"
)

type MinIODriver struct {
	S3Config
	minioClient *MinIOClient
	kubeClient  *kubernetes.Clientset
}

func NewMinIODriver(config *S3Config) (*MinIODriver, error) {
	minioClient, err := NewMinIOClient(config)
	if err != nil {
		return nil, err
	}

	k8sCfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	kubeClient, err := kubernetes.NewForConfig(k8sCfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	return &MinIODriver{
		S3Config:    *config,
		minioClient: minioClient,
		kubeClient:  kubeClient,
	}, nil
}

func (driver *MinIODriver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	volumeParam := req.GetParameters()
	bucketName := req.GetName()
	// pvc info
	pvcName := volumeParam[ParamPVCName]
	pvcNamespace := volumeParam[ParamPVCNameSpace]
	if pvcName == "" || pvcNamespace == "" {
		return nil, status.Errorf(codes.InvalidArgument, "CreateVolume: pvcName(%s) or pvcNamespace(%s) is empty", pvcName, pvcNamespace)
	} else {
		pvc, err := driver.kubeClient.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(ctx, pvcName, metav1.GetOptions{})
		if err != nil {
			return &csi.CreateVolumeResponse{}, err
		}
		anno := pvc.GetAnnotations()
		if name, exist := anno[AnnoBucketName]; exist {
			bucketName = name
		}
	}

	capacity := req.GetCapacityRange().RequiredBytes
	if err := driver.minioClient.CreateBucket(bucketName, capacity); err != nil {
		return &csi.CreateVolumeResponse{}, status.Error(codes.Internal, err.Error())
	}

	volumeParam[ParamProvisionTypeTag] = string(ProvisionTypeBucketOrCreate)
	volumeParam[ParamBucketNameTag] = bucketName
	volumeParam[common.ParamDriverName] = DriverName

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      req.GetName(),
			CapacityBytes: capacity,
			VolumeContext: volumeParam,
		},
	}, nil
}

func (driver *MinIODriver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	pv, err := driver.kubeClient.CoreV1().PersistentVolumes().Get(ctx, req.GetVolumeId(), metav1.GetOptions{})
	if err != nil {
		return &csi.DeleteVolumeResponse{}, status.Error(codes.Internal, err.Error())
	}

	bucketName := pv.Spec.CSI.VolumeAttributes[ParamBucketNameTag]
	if err := driver.minioClient.DeleteBucket(bucketName); err != nil {
		return &csi.DeleteVolumeResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (driver *MinIODriver) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	pv, err := driver.kubeClient.CoreV1().PersistentVolumes().Get(ctx, req.GetVolumeId(), metav1.GetOptions{})
	if err != nil {
		return &csi.ControllerExpandVolumeResponse{}, status.Error(codes.Internal, err.Error())
	}

	bucketName := pv.Spec.CSI.VolumeAttributes[ParamBucketNameTag]
	capacity := req.GetCapacityRange().RequiredBytes
	if DefaultFeatureGate.Enabled(Quota) {
		if err := driver.minioClient.SetBucketQuota(bucketName, capacity, madmin.HardQuota); err != nil {
			return &csi.ControllerExpandVolumeResponse{}, status.Error(codes.Internal, err.Error())
		}
	}

	bucketMap, err := driver.minioClient.GetBucketMetadata(bucketName)
	if err != nil {
		return &csi.ControllerExpandVolumeResponse{}, status.Error(codes.Internal, err.Error())
	}
	bucketMap[MetaDataCapacity] = strconv.FormatInt(capacity, 10)
	if err = driver.minioClient.SetBucketMetadata(bucketName, bucketMap); err != nil {
		return &csi.ControllerExpandVolumeResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &csi.ControllerExpandVolumeResponse{CapacityBytes: capacity, NodeExpansionRequired: false}, nil
}

func (driver *MinIODriver) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return &csi.NodeExpandVolumeResponse{}, status.Error(codes.Unimplemented, "NodeExpandVolume is not implemented")
}

func (driver *MinIODriver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return &csi.NodeStageVolumeResponse{}, status.Error(codes.Unimplemented, "NodeStageVolume is not implemented")
}

func (driver *MinIODriver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return &csi.NodeUnstageVolumeResponse{}, status.Error(codes.Unimplemented, "NodeUnstageVolume is not implemented")
}

func (driver *MinIODriver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	pv, err := driver.kubeClient.CoreV1().PersistentVolumes().Get(ctx, req.GetVolumeId(), metav1.GetOptions{})
	if err != nil {
		return &csi.NodePublishVolumeResponse{}, status.Error(codes.Internal, err.Error())
	}
	bucketName := pv.Spec.CSI.VolumeAttributes[ParamBucketNameTag]
	targetPath := req.GetTargetPath()

	notMnt, err := checkMount(targetPath)
	if err != nil {
		return &csi.NodePublishVolumeResponse{}, status.Error(codes.Internal, err.Error())
	}
	if !notMnt {
		return &csi.NodePublishVolumeResponse{}, nil
	}

	if err := S3FSMount(driver.Endpoint, bucketName, targetPath, driver.AK, driver.SK); err != nil {
		return &csi.NodePublishVolumeResponse{}, err
	}

	klog.Infof("s3: bucket %s successfully mounted to %s", bucketName, targetPath)
	return &csi.NodePublishVolumeResponse{}, nil

}

func (driver *MinIODriver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()

	// Check arguments
	if len(volumeID) == 0 {
		return &csi.NodeUnpublishVolumeResponse{}, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(targetPath) == 0 {
		return &csi.NodeUnpublishVolumeResponse{}, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	mountPoint := req.GetTargetPath()
	if !isS3fsMounted(mountPoint) {
		klog.Infof("Directory is not mounted: %s", mountPoint)
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	if err := S3FSUmount(targetPath); err != nil {
		return &csi.NodeUnpublishVolumeResponse{}, err
	}
	klog.Infof("s3: mountpoint %s has been unmounted.", targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (driver *MinIODriver) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	var err error
	// volumeID is bucket name, and pv name too
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return &csi.NodeGetVolumeStatsResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("NodeGetVolumeStats target local path %v is empty", volumeID))
	}

	pv, err := driver.kubeClient.CoreV1().PersistentVolumes().Get(ctx, req.GetVolumeId(), metav1.GetOptions{})
	if err != nil {
		return &csi.NodeGetVolumeStatsResponse{}, status.Error(codes.Internal, err.Error())
	}
	bucketName := pv.Spec.CSI.VolumeAttributes[ParamBucketNameTag]
	available, capacity, usage, inodes, inodesFree, inodesUsed, err := driver.minioClient.FsInfo(bucketName)
	if err != nil {
		return &csi.NodeGetVolumeStatsResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			{
				Available: available,
				Total:     capacity,
				Used:      usage,
				Unit:      csi.VolumeUsage_BYTES,
			},
			{
				Available: inodesFree,
				Total:     inodes,
				Used:      inodesUsed,
				Unit:      csi.VolumeUsage_INODES,
			},
		},
	}, nil
}

func checkMount(targetPath string) (bool, error) {
	notMnt, err := mount.New("").IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(targetPath, 0750); err != nil {
				return false, err
			}
			notMnt = true
		} else {
			return false, err
		}
	}

	return notMnt, nil
}

func isS3fsMounted(mountPath string) bool {
	checkMountCountCmd := fmt.Sprintf("%s mount | grep %s | grep %s | grep -v grep | wc -l", common.NsenterCmd, mountPath, S3FSType)
	out, err := common.RunCommand(checkMountCountCmd)
	if err != nil {
		return false
	}
	if strings.TrimSpace(out) == "0" {
		return false
	}
	return true
}
