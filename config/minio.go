package config

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
)

func (cfg Config) NewMinio() (*minio.Client, error) {
	minioClient, err := minio.New(cfg.Storage.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Storage.AccessKey, cfg.Storage.SecretKey, ""),
		Secure: cfg.Storage.UseSSL,
	})

	if err != nil {
		log.Error().
			Err(err).
			Str("source", "config.NewMinio").
			Msg("Failed connect to minio storage")
	}

	ctx := context.Background()

	// test koneksi
	_, err = minioClient.ListBuckets(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "config.NewMinio").
			Msg("Failed to get list bukcet")
	}

	// nama bucket dari config
	bucketName := cfg.Storage.Bucket

	// cek bucket sudah ada atau belum
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "config.NewMinio").
			Msg("Failed to get list bukcet")
	}

	// kalau belum ada → buat bucket
	if !exists {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Error().
				Err(err).
				Str("source", "config.NewMinio").
				Msg("Failed to get list bukcet")
		}
	}

	// cek existing policy
	existingPolicy, err := minioClient.GetBucketPolicy(ctx, bucketName)
	if err != nil {
		// kalau error selain "no policy", tetap fail
		log.Error().
			Err(err).
			Str("source", "config.NewMinio").
			Msg("Failed to get policy")
		return nil, err
	}

	// kalau belum ada policy → set
	// - semua orang boleh download / lihat file
	// - tidak boleh upload, delete, overwrite
	if existingPolicy == "" {
		policy := fmt.Sprintf(`{
			"Version":"2012-10-17",
			"Statement":[
				{
					"Effect":"Allow",
					"Principal":"*",
					"Action":["s3:GetObject"],
					"Resource":["arn:aws:s3:::%s/*"]
				}
			]
		}`, bucketName)

		if err := minioClient.SetBucketPolicy(ctx, bucketName, policy); err != nil {
			log.Error().
				Err(err).
				Str("source", "config.NewMinio").
				Msg("Failed to set policy")
			return nil, err
		}
	}

	return minioClient, nil
}
