package pipeline

func recordExpiredCleanupRemoval(result *cleanupResult, size int64) {
	result.removed++
	result.removedTTL++
	result.removedBytes += size
	result.removedTTLBytes += size
	result.totalBytes -= size
}

func recordSizeCleanupRemoval(result *cleanupResult, size int64) {
	result.removed++
	result.removedSize++
	result.removedBytes += size
	result.removedSizeBytes += size
	result.totalBytes -= size
}
