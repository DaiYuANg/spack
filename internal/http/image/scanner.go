package image

import (
	"go.etcd.io/bbolt"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"sproxy/internal/config"
)

const bucketName = "Meta"

var scanSupportedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

type ScannerDependency struct {
	fx.In
	DB     *bbolt.DB
	Config *config.Config
	Logger *zap.SugaredLogger
}

func prepareScan(db *bbolt.DB, logger *zap.SugaredLogger) {
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		logger.Fatalf("failed to create bucket: %v", err)
	}
}

//func scan(dep ScannerDependency) {
//	db, cfg, logger := dep.DB, dep.Config, dep.Logger
//	err := filepath.Walk(cfg.Spa.Static, func(path string, info os.FileInfo, err error) error {
//		if err != nil {
//			logger.Errorf("skip %s due to error: %v", path, err)
//			return nil
//		}
//
//		if info.IsDir() {
//			return nil
//		}
//
//		ext := filepath.Ext(path)
//		if !scanSupportedExts[ext] {
//			return nil
//		}
//
//		meta, err := scanFile(path, info)
//		if err != nil {
//			logger.Errorf("failed to scan %s: %v", path, err)
//			return nil
//		}
//
//		// 写入 BoltDB
//		err = db.Update(func(tx *bbolt.Tx) error {
//			b := tx.Bucket([]byte(bucketName))
//			data, err := json.Marshal(meta)
//			if err != nil {
//				return err
//			}
//			return b.Put([]byte(meta.Path), data)
//		})
//		if err != nil {
//			logger.Errorf("failed to save meta for %s: %v", path, err)
//		} else {
//			logger.Debugf("scanned and saved: %s", path)
//		}
//		return nil
//	})
//	if err != nil {
//		logger.Fatalf("walk error: %v", err)
//	}
//}
//
//func scanFile(path string, info os.FileInfo) (*Meta, error) {
//	f, err := os.Open(path)
//	if err != nil {
//		return nil, err
//	}
//	defer func(f *os.File) {
//		err := f.Close()
//		if err != nil {
//			panic(err)
//		}
//	}(f)
//
//	hasher := sha256.New()
//	_, err = io.Copy(hasher, f)
//	if err != nil {
//		return nil, err
//	}
//
//	return &Meta{
//		Path:        path,
//		Size:        info.Size(),
//		ModTime:     info.ModTime(),
//		ContentHash: hex.EncodeToString(hasher.Sum(nil)),
//	}, nil
//}
