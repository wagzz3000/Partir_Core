package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	// Postgres Config
	dbURL := "postgres://partir:partir@localhost:5432/partir?sslmode=disable"
	if url := os.Getenv("DATABASE_URL"); url != "" {
		dbURL = url
	}

	// MinIO Config
	minioEndpoint := "localhost:9000"
	minioAccessKey := "partir"
	minioSecretKey := "partirpass"
	minioBucket := "partir-artifacts"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Check Postgres
	fmt.Print("Checking Postgres connectivity... ")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("FAILED: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("FAILED: ping error: %v", err)
	}
	fmt.Println("OK")

	// 2. Check MinIO
	fmt.Print("Checking MinIO connectivity... ")
	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccessKey, minioSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("FAILED: %v", err)
	}

	// List buckets to verify auth
	buckets, err := minioClient.ListBuckets(ctx)
	if err != nil {
		log.Fatalf("FAILED: list buckets error: %v", err)
	}

	found := false
	for _, b := range buckets {
		if b.Name == minioBucket {
			found = true
			break
		}
	}

	if found {
		fmt.Println("OK (bucket found)")
	} else {
		fmt.Printf("OK (bucket %q not found, creating...)\n", minioBucket)
		err = minioClient.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("FAILED: create bucket error: %v", err)
		}
		fmt.Println("Created bucket")
	}

	fmt.Println("\nStack verification SUCCEEDED!")
}
