package tasks

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/thornhall/blog/internal/backup"
)

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
