package main

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

// generatePresignedURL generates a temporary, signed URL for accessing an S3
// object identified by bucket and key. The URL expires after expireTime.
func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	// NewPresignClient wraps our existing s3Client, reusing its config/credentials
	// rather than requiring a separate S3 client setup just for presigning.
	presignClient := s3.NewPresignClient(s3Client)

	// This is used to identify the object being retrieved.
	getObjectParams := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	// Functional option WithPresignExpires allows us to set an expiry on the URL we will generate.
	presignedHTTPReq, err := presignClient.PresignGetObject(context.TODO(), getObjectParams, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}

	return presignedHTTPReq.URL, nil
}

// dbVideoToSignedVideo takes a database.Video as input and returns a new database.Video
// with the same metadata as the original, except the VideoURL field is set to a presigned URL.
// Also returns an error (returned by the handler).
func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	// This protects against a nil pointer dereference in strings.Split
	// in the case where the video doesn't have a URL set in the database.
	if video.VideoURL == nil {
		// We don't return an error here so that when we loop through the retrieved videos in handlerVideosRetrieve
		// to generate a signed URL for each video, we don't error out upon hitting a nil URL and stop the loop from iterating
		// through the remaining videos in the list which may have URLs and can therefore be overwritten with a presigned URL
		return video, nil
	}
	bucketAndKey := strings.Split(*video.VideoURL, ",")
	bucket := bucketAndKey[0]
	key := bucketAndKey[1]

	// We use a 2 minute expiry because the lesson emphasizes short-lived
	// signed URLs for security. The frontend sets this URL once as the
	// <video> element's src and never refreshes it, so if a viewer seeks
	// into an unbuffered part of a longer video after the URL expires,
	// the <video> element's own range request for that data will fail.
	// This is the security/convenience tradeoff we're accepting.
	preSignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, 2*time.Minute)
	if err != nil {
		return video, err
	}

	video.VideoURL = &preSignedURL

	return video, nil
}
