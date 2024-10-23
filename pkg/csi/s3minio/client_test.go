package s3minio

import (
	"context"
	"github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestNewMinIOClient(t *testing.T) {
	mockey.PatchConvey("test NewMinIOClient", t, func() {
		cfg := S3Config{
			AK:       "NvmbqoUxn50jqcHlHBEG",
			SK:       "EAGogws8UcIA8WuBMOuGDRKIC7MXvWcswD3dHpzW",
			Endpoint: "http://10.31.25.164:32240",
			Region:   "china",
		}
		ctx := context.Background()

		c, err := NewMinIOClient(&cfg)
		convey.So(err, convey.ShouldBeNil)
		buckets, err := c.mclient.ListBuckets(ctx)
		convey.So(err, convey.ShouldBeNil)
		convey.So(buckets, convey.ShouldNotBeNil)
		ok, err := c.mclient.BucketExists(ctx, "fuse-3bcc7b3d-c8c3-418b-85f6-08b083faa329")
		convey.So(err, convey.ShouldBeNil)
		convey.So(ok, convey.ShouldBeFalse)
	})
}
