---
title: Improving Durability of the Blog
slug: improving-durability
date: Nov 22, 2025
category: Engineering
excerpt: What to do in case of hardware failures.
---
I've mentioned previously that my database is SQLite, an embedded database.

That means the database runs on the same machine as my code. It also means that if my code were to lose access
to the physical storage of the machine it's running on somehow, we could lose our DB.

To account for this, I deployed my app using a Digital Ocean VPS. A VPS has some capabilities that make it more suitable for
persistent storage like this. In case of reboots or a physical drive failure, data will be safe.

However, there are still scenarios where my SQLite DB could be wiped:
- Destruction of the instance
- Rebuild of the instance
- (less likely) a data center fire

Because of this, I decided to implement a simple backup mechanism for the blog's db.

## The Backup Method
Our method and implementation is simple - periodically, the app will write the DB to cloud storage, in this case S3.
This makes our backups **automated**. In the case that my SQLite DB is lost, I can simply copy the backup from S3 back to my VPS.

## Implementation
To implement this, we need a way to interface with object storage like S3. Thankfully, there are libraries that do this for us.

First, we create a client that knows how to interact with cloud storage like S3.

```go
func NewSpaceClient() (*SpaceClient, error) {
	if os.Getenv("ENV") != "prod" {
		return nil, fmt.Errorf("environment is not prod - not configuring backups")
	}
	key := os.Getenv("SPACES_KEY")
	secret := os.Getenv("SPACES_SECRET")
	endpoint := os.Getenv("SPACES_ENDPOINT")
	region := os.Getenv("SPACES_REGION")
	bucket := os.Getenv("SPACES_BUCKET")

	creds := credentials.NewStaticCredentialsProvider(key, secret, "")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &SpaceClient{
		Client: client,
		Bucket: bucket,
	}, nil
}
```

This struct has a method that knows how to upload a file to the cloud storage:

```go
func (s *SpaceClient) UploadFile(ctx context.Context, objectKey string, fileReader io.Reader) error {
	_, err := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(objectKey),
		Body:   fileReader,
		ACL:    "private",
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	return nil
}
```

Now, we need a worker that will periodically call this `UploadFile` method. A goroutine works perfectly for this.

```go
type BackupService struct {
	spaceClient *backup.SpaceClient
	dbPath      string
	interval    time.Duration
}

func NewBackupService(client *backup.SpaceClient, dbPath string, interval time.Duration) *BackupService {
	return &BackupService{
		spaceClient: client,
		dbPath:      dbPath,
		interval:    interval,
	}
}

func (b *BackupService) Start(ctx context.Context) {
	ticker := time.NewTicker(b.interval)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				if err := b.performBackup(ctx); err != nil {
					log.Printf("Backup failed: %v", err)
				} else {
					log.Printf("Backup successful")
				}
			}
		}
	}()
}

func (b *BackupService) performBackup(ctx context.Context) error {
	f, err := os.Open(b.dbPath)
	if err != nil {
		return err
	}
	defer f.Close()

	filename := "backups/blog.db"

	return b.spaceClient.UploadFile(ctx, filename, f)
}
```

Given the interval, in the `Start` method, we create a new goroutine that wakes up every `interval` seconds and uploads the backup, via this block:
```go
case <-ticker.C:
	if err := b.performBackup(ctx); err != nil {
		log.Printf("Backup failed: %v", err)
	} else {
		log.Printf("Backup successful")
	}
}
```

Now, we need to wire the whole thing up into our `main.go` file, so that it starts the worker when the server is started.

```go
	package main

	backupCtx, cancelBackup := context.WithCancel(context.Background())
	defer cancelBackup()

	backupClient, err := backup.NewSpaceClient()
	if err != nil {
		log.Printf("error getting S3 client: %v", err)
	} else {
		backupWorker := tasks.NewBackupService(backupClient, "blog.db", time.Hour)
		backupWorker.Start(backupCtx)
	}
```

This code executes just after we start the actual server.

## Conclusion
Now our blog has automated backups, and the VPS is no longer a single point of failure.
