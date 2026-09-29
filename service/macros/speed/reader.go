package speed

import "sync"

// ReadCounter 上传计数器（对应下载的WriteCounter）
type ReadCounter struct {
	mu        sync.Mutex
	count     uint64
	RateLimit int64
}

func (rc *ReadCounter) Write(p []byte) (n int, err error) {
	rc.mu.Lock()
	rc.count += uint64(len(p))
	rc.mu.Unlock()
	return len(p), nil
}

func (rc *ReadCounter) Take() uint64 {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	count := rc.count
	rc.count = 0
	return count
}
