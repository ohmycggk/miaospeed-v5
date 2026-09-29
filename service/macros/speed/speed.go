package speed

import (
	"context"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/juju/ratelimit"

	"github.com/miaokobot/miaospeed/interfaces"
	"github.com/miaokobot/miaospeed/preconfigs"
	"github.com/miaokobot/miaospeed/utils"
	"github.com/miaokobot/miaospeed/utils/structs"
	"github.com/miaokobot/miaospeed/vendors"
)

func Once(speed *Speed, proxy interfaces.Vendor, cfg *interfaces.SlaveRequestConfigs) {
	speed.Speeds = make([]uint64, cfg.DownloadDuration)

	downloadFiles := RefetchDownloadFiles(proxy, cfg.DownloadURL)
	utils.DLogf("Speed Prefetch | Using files arr=%v", downloadFiles)

	th := int(cfg.DownloadThreading)
	wcGroups := []*WriteCounter{}
	ctxCancels := []context.CancelFunc{}

	initWG := sync.WaitGroup{}
	writingLock := sync.Mutex{}
	for i := 0; i < th; i++ {
		initWG.Add(1)
		go func() {
			wc := WriteCounter{
				RateLimit: int64(utils.GCFG.SpeedLimit) / int64(th),
			}
			cancelFunc := SingleThread(downloadFiles, proxy, cfg.DownloadDuration, &wc)

			writingLock.Lock()
			ctxCancels = append(ctxCancels, cancelFunc)
			wcGroups = append(wcGroups, &wc)
			writingLock.Unlock()

			initWG.Done()
		}()
	}
	initWG.Wait()

	// normalization
	for i := 0; i < th; i++ {
		wcGroups[i].Take()
	}

	for t := 0; t < int(cfg.DownloadDuration); t++ {
		time.Sleep(time.Second - time.Millisecond*10)
		byteLen := uint64(0)
		for i := 0; i < th; i++ {
			threadLen := wcGroups[i].Take()
			utils.DLogf("Task Thread | time=%d thread=%d speed=%d", t+1, i+1, threadLen)
			byteLen += threadLen
		}
		speed.Speeds[t] = byteLen
		speed.TotalSize += byteLen
		speed.MaxSpeed = structs.Max(speed.MaxSpeed, byteLen)
	}
	speed.AvgSpeed = speed.TotalSize / uint64(cfg.DownloadDuration)

	for i := 0; i < th; i++ {
		ctxCancels[i]()
	}
}

func SingleThread(downloadFiles []string, proxy interfaces.Vendor, timeoutSeconds int64, wc *WriteCounter) context.CancelFunc {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds+1)*time.Second)
	isCancelled := false

	downloadFilesCopy := downloadFiles[:]
	fileLen := len(downloadFilesCopy)
	readyChan := make(chan bool)

	go func() {
		isReady := false
		defer func() {
			if !isReady {
				close(readyChan)
			}
		}()

		// 100 only for safty
		for i := 0; i < 100; i++ {
			// if outside cancel or deadline meet(either by time or by hand)
			if isCancelled || ctx.Err() != nil {
				return
			}
			// download file
			file := downloadFilesCopy[i%fileLen]
			resp, _, err := vendors.RequestUnsafe(ctx, proxy, &interfaces.RequestOptions{
				URL: file,
			})

			if !isReady {
				isReady = true
				close(readyChan)
			}
			if err == nil {
				var bodyReader io.Reader = nil
				if wc.RateLimit >= 1024 {
					bucket := ratelimit.NewBucketWithRate(float64(wc.RateLimit)*0.95, wc.RateLimit)
					bodyReader = ratelimit.Reader(resp.Body, bucket)
				} else {
					bodyReader = resp.Body
				}
				io.Copy(ioutil.Discard, io.TeeReader(bodyReader, wc))
			}
			// close body
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
		}
	}()

	<-readyChan
	return func() {
		isCancelled = true
		cancel()
	}
}

func RefetchDownloadFiles(proxy interfaces.Vendor, file string) []string {
	defaultList := []string{preconfigs.SPEED_DEFAULT_LARGE_FILE_STATIC_MSFT}
	if proxy == nil || proxy.Status() == interfaces.VStatusNotReady {
		return defaultList
	}

	switch file {
	case preconfigs.SPEED_DEFAULT_LARGE_FILE_DYN_INTL:
		body, _, _ := vendors.RequestWithRetry(proxy, 1, 1000, &interfaces.RequestOptions{
			URL:     "https://ipinfo.io",
			NoRedir: true,
		})

		if strings.Contains(string(body), "Microsoft") {
			return []string{preconfigs.SPEED_DEFAULT_LARGE_FILE_STATIC_MSFT}
		} else {
			return []string{preconfigs.SPEED_DEFAULT_LARGE_FILE_STATIC_GOOGLE}
		}
	case preconfigs.SPEED_DEFAULT_LARGE_FILE_DYN_FAST:
		body, _, _ := vendors.RequestWithRetry(proxy, 3, 1000, &interfaces.RequestOptions{
			URL:     "https://api.fast.com/netflix/speedtest/v2?https=false&token=YXNkZmFzZGxmbnNkYWZoYXNkZmhrYWxm&urlCount=5",
			NoRedir: true,
		})
		url := jsoniter.Get(body, "targets", 0, "url").ToString()
		if url != "" {
			return []string{url}
		} else {
			return defaultList
		}
	}
	return []string{file}
}

func RefetchUploadURL(proxy interfaces.Vendor, uploadURL string) string {
	defaultURL := preconfigs.SPEED_DEFAULT_UPLOAD_URL
	if proxy == nil || proxy.Status() == interfaces.VStatusNotReady {
		return defaultURL
	}

	if uploadURL == preconfigs.SPEED_DEFAULT_UPLOAD_URL_DYN || uploadURL == "" {
		return defaultURL
	}
	return uploadURL
}

func normalizeUpload(cfg *interfaces.SlaveRequestConfigs) (int64, int) {
	duration := cfg.UploadDuration
	if duration < 1 || duration > 30 {
		duration = preconfigs.SPEED_DEFAULT_DURATION
	}
	threading := cfg.UploadThreading
	if threading < 1 || threading > 32 {
		threading = preconfigs.SPEED_DEFAULT_THREADING
	}
	return duration, int(threading)
}

// UploadOnceChunked measures upload throughput with chunked POST bodies.
func UploadOnceChunked(speed *UploadSpeed, proxy interfaces.Vendor, cfg *interfaces.SlaveRequestConfigs) {
	if !utils.GCFG.EnableUploadSpeedFlag || cfg.ApiVersion < interfaces.ApiV3 {
		return
	}

	duration, th := normalizeUpload(cfg)
	speed.Speeds = make([]uint64, duration)
	uploadURL := RefetchUploadURL(proxy, cfg.UploadURL)
	utils.DLogf("Upload Speed (chunked) | Using upload URL=%v", uploadURL)

	var rcGroups []*ReadCounter
	var ctxCancels []context.CancelFunc

	initWG := sync.WaitGroup{}
	writingLock := sync.Mutex{}
	for i := 0; i < th; i++ {
		initWG.Add(1)
		go func() {
			limit := int64(0)
			if th > 0 {
				limit = int64(utils.GCFG.SpeedLimit) / int64(th)
			}
			rc := ReadCounter{RateLimit: limit}
			cancelFunc := SingleUploadThreadChunked(uploadURL, proxy, duration, &rc)

			writingLock.Lock()
			ctxCancels = append(ctxCancels, cancelFunc)
			rcGroups = append(rcGroups, &rc)
			writingLock.Unlock()

			initWG.Done()
		}()
	}
	initWG.Wait()

	for i := 0; i < len(rcGroups); i++ {
		rcGroups[i].Take()
	}

	for t := 0; t < int(duration); t++ {
		time.Sleep(time.Second - time.Millisecond*10)
		byteLen := uint64(0)
		for i := 0; i < len(rcGroups); i++ {
			threadLen := rcGroups[i].Take()
			utils.DLogf("Upload Chunked Task | time=%d thread=%d speed=%d", t+1, i+1, threadLen)
			byteLen += threadLen
		}
		speed.Speeds[t] = byteLen
		speed.TotalSize += byteLen
		speed.MaxSpeed = structs.Max(speed.MaxSpeed, byteLen)
	}
	speed.AvgSpeed = speed.TotalSize / uint64(duration)

	for i := 0; i < len(ctxCancels); i++ {
		ctxCancels[i]()
	}
}

func SingleUploadThreadChunked(uploadURL string, proxy interfaces.Vendor, timeoutSeconds int64, rc *ReadCounter) context.CancelFunc {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds+3)*time.Second)
	isCancelled := false
	readyChan := make(chan bool)

	go func() {
		isReady := false
		defer func() {
			if !isReady {
				close(readyChan)
			}
		}()

		chunkSize := 64 * 1024 * 2
		pattern := []byte(strings.Repeat("0", chunkSize))

		for i := 0; i < 100; i++ {
			if isCancelled || ctx.Err() != nil {
				return
			}
			pr, pw := io.Pipe()
			var wg sync.WaitGroup
			writerStopped := make(chan struct{})

			mw := io.MultiWriter(pw, rc)
			wg.Add(1)
			go func() {
				defer func() {
					wg.Done()
					close(writerStopped)
				}()

				ticker := time.NewTicker(time.Millisecond * 10)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						_ = pw.Close()
						return
					case <-ticker.C:
						select {
						case <-writerStopped:
							return
						default:
						}

						_, err := mw.Write(pattern)
						if err != nil {
							_ = pw.Close()
							return
						}
					}
				}
			}()

			requestTimeout := timeoutSeconds / 2
			if requestTimeout < 1 {
				requestTimeout = 1
			}
			requestCtx, requestCancel := context.WithTimeout(ctx, time.Duration(requestTimeout)*time.Second)

			resp, _, err := vendors.RequestUnsafe(requestCtx, proxy, &interfaces.RequestOptions{
				URL:    uploadURL,
				Method: "POST",
				Reader: pr,
				Headers: map[string]string{
					"Content-Type":      "application/octet-stream",
					"Transfer-Encoding": "chunked",
				},
			})

			if !isReady {
				isReady = true
				close(readyChan)
			}

			requestCancel()
			_ = pw.Close()
			wg.Wait()

			if err != nil {
				if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "EOF") {
					utils.DErrorf("Upload Chunked Task | Request error: %v", err)
				}
			} else if resp != nil && resp.Body != nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}

			if isCancelled || ctx.Err() != nil {
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
	}()

	<-readyChan
	return func() {
		isCancelled = true
		cancel()
	}
}
