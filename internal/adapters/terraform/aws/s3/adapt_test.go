package s3

import (
	"testing"

	"github.com/aquasecurity/defsec/internal/adapters/terraform/tftestutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PublicAccessBlock(t *testing.T) {
	testCases := []struct {
		desc            string
		source          string
		expectedBuckets int
		hasPublicAccess bool
	}{
		{
			desc: "public access block is found when using the bucket name as the lookup",
			source: `
resource "aws_s3_bucket" "example" {
	bucket = "bucketname"
}

resource "aws_s3_bucket_public_access_block" "example_access_block"{
	bucket = "bucketname"
}
`,
			expectedBuckets: 1,
			hasPublicAccess: true,
		},
		{
			desc: "public access block is found when using the bucket name as the lookup",
			source: `
resource "aws_s3_bucket" "example" {
	bucket = "bucketname"
}

resource "aws_s3_bucket_public_access_block" "example_access_block"{
	bucket = aws_s3_bucket.example.id
}
`,
			expectedBuckets: 1,
			hasPublicAccess: true,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {

			modules := tftestutil.CreateModulesFromSource(t, tC.source, ".tf")
			s3Ctx := Adapt(modules)

			assert.Equal(t, tC.expectedBuckets, len(s3Ctx.Buckets))

			for _, bucket := range s3Ctx.Buckets {
				if tC.hasPublicAccess {
					assert.NotNil(t, bucket.PublicAccessBlock)
				} else {
					assert.Nil(t, bucket.PublicAccessBlock)
				}
			}

			bucket := s3Ctx.Buckets[0]
			assert.NotNil(t, bucket.PublicAccessBlock)

		})
	}

}

func Test_PublicAccessDoesNotReference(t *testing.T) {
	testCases := []struct {
		desc   string
		source string
	}{
		{
			desc: "just a bucket, no public access block",
			source: `
resource "aws_s3_bucket" "example" {
	bucket = "bucketname"
}
			`,
		},
		{
			desc: "bucket with unrelated public access block",
			source: `
resource "aws_s3_bucket" "example" {
	bucket = "bucketname"
}

resource "aws_s3_bucket_public_access_block" "example_access_block"{
	bucket = aws_s3_bucket.other.id
}
			`,
		},
		{
			desc: "bucket with unrelated public access block via name",
			source: `
resource "aws_s3_bucket" "example" {
	bucket = "bucketname"
}

resource "aws_s3_bucket_public_access_block" "example_access_block"{
	bucket = "something"
}
			`,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			modules := tftestutil.CreateModulesFromSource(t, tC.source, ".tf")
			s3Ctx := Adapt(modules)
			require.Len(t, s3Ctx.Buckets, 1)
			assert.Nil(t, s3Ctx.Buckets[0].PublicAccessBlock)

		})
	}
}
