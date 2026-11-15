package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

var httpClient *http.Client

func initHttpClient(preferIPv6 bool) {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 3 * time.Second,
			}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			ips, err := resolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}

			for _, ip := range ips {
				if preferIPv6 {
					if ip.To4() == nil {
						return dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), port))
					}
				} else {
					if ip.To4() != nil {
						return dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), port))
					}
				}
			}

			return nil, fmt.Errorf("no suitable IP found for %s", host)
		},
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}

	httpClient = &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}
}

func checkNetworkStats(preferIPv6 bool) {
	fmt.Printf("\n%s Determining network quality to adjust scan options...\n\n", prompt)
	const (
		testTargetURL     = "http://www.google.com/generate_204"
		initialTestCount  = 100
		goodLatencyMs     = 50
		moderateLatencyMs = 100
		poorLatencyMs     = 200
		acceptableLoss    = 5.0
		highLoss          = 10.0
		moderateJitterMs  = 5.0
		highJitterMs      = 10.0
		maxConcurrency    = 5
	)

	var (
		totalLatency       int64
		successCount       int
		wg                 sync.WaitGroup
		latencyResults     = make(chan int64, initialTestCount)
		concurrencyLimiter = make(chan struct{}, maxConcurrency)
	)

	networkMode := "IPv4"
	if preferIPv6 {
		networkMode = "IPv6"
	}
	desc := fmt.Sprintf("Testing %s network...", networkMode)
	bar := progressbar.NewOptions(initialTestCount,
		progressbar.OptionShowBytes(false),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]#[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	initHttpClient(preferIPv6)
	for range initialTestCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			concurrencyLimiter <- struct{}{}
			defer func() { <-concurrencyLimiter }()
			start := time.Now()
			resp, err := httpClient.Head(testTargetURL)
			bar.Add(1)
			latency := time.Since(start).Milliseconds()
			if err == nil && resp.StatusCode == http.StatusNoContent {
				if resp.Body != nil {
					resp.Body.Close()
				}
				latencyResults <- latency
			} else {
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
				latencyResults <- -1
			}
		}()
	}

	wg.Wait()
	close(latencyResults)
	fmt.Println()

	successfulLatencies := make([]int64, 0, successCount)
	for res := range latencyResults {
		if res >= 0 {
			successCount++
			totalLatency += res
			successfulLatencies = append(successfulLatencies, res)
		}
	}

	if successCount == 0 {
		failMessage("Initial network quality test failed. Could not reach test server.")
		fmt.Printf("\n%s Fallback to default scan settings.\n", prompt)
		return
	}

	slices.Sort(successfulLatencies)
	medianLatency := successfulLatencies[len(successfulLatencies)/2]
	lossRate := float64(initialTestCount-successCount) / float64(initialTestCount) * 100
	var avgJitter float64

	if successCount > 1 {
		var totalJitter int64
		for i := 0; i < len(successfulLatencies)-1; i++ {
			diff := successfulLatencies[i+1] - successfulLatencies[i]
			if diff < 0 {
				diff = -diff
			}
			totalJitter += diff
		}
		avgJitter = float64(totalJitter) / float64(successCount-1)
		fmt.Printf("\n%s Avg Latency: %dms | Jitter: %.1fms | Loss: %.1f%%\n", prompt, medianLatency, avgJitter, lossRate)
	} else {
		fmt.Printf("\n%s Avg Latency: %dms | Jitter not calculated (unsuccessful tests) | Loss: %.1f%%\n", prompt, medianLatency, lossRate)
	}

	if medianLatency >= int64(poorLatencyMs) || lossRate >= highLoss || (successCount > 1 && avgJitter >= highJitterMs) {
		if preferIPv6 {
			scanConfig.IPv6Retries = 7
		} else {
			scanConfig.IPv4Retries = 7
		}
		successMessage("Network appears slow/unstable.")
	} else if medianLatency >= int64(moderateLatencyMs) || lossRate >= acceptableLoss || (successCount > 1 && avgJitter >= moderateJitterMs) {
		if preferIPv6 {
			scanConfig.IPv6Retries = 5
		} else {
			scanConfig.IPv4Retries = 5
		}
		successMessage("Network is moderate or some packet loss detected.")
	} else {
		successMessage("Network quality seems good. Using default scan settings.")
	}
}

func scanEndpoints() ([]ScanResult, error) {
	if len(scanConfig.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints were generated")
	}
	if scanConfig.IPv4Retries < 1 && scanConfig.IPv6Retries < 1 {
		return nil, fmt.Errorf("retry count must be at least one")
	}

	// One WARP registration is shared by all batches. Xray only needs to hold
	// a limited number of local inbounds at once, even during a Deep scan.
	warpParams, err := getWarpParams()
	if err != nil {
		return nil, err
	}
	allEndpoints := scanConfig.Endpoints
	activeWarpParams = &warpParams
	defer func() {
		scanConfig.Endpoints = allEndpoints
		activeWarpParams = nil
	}()

	const batchSize = 256
	allResults := make([]ScanResult, 0, len(allEndpoints))
	for offset := 0; offset < len(allEndpoints); offset += batchSize {
		end := min(offset+batchSize, len(allEndpoints))
		scanConfig.Endpoints = allEndpoints[offset:end]
		fmt.Printf("\n%s Scanning batch %d/%d (%d endpoints)\n",
			prompt, offset/batchSize+1, (len(allEndpoints)+batchSize-1)/batchSize, end-offset)

		batchResults, err := scanEndpointBatch()
		if err != nil {
			return nil, fmt.Errorf("scan batch %d failed: %w", offset/batchSize+1, err)
		}
		allResults = append(allResults, batchResults...)
	}
	return allResults, nil
}

func scanEndpointBatch() ([]ScanResult, error) {
	if err := createXrayConfig(); err != nil {
		return nil, err
	}

	cmd, err := runXrayCore()
	if err != nil {
		return nil, err
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	// A bounded worker pool avoids creating thousands of sleeping goroutines
	// during Deep scans. Retries run inside each worker, so concurrency is
	// bounded across the whole scan rather than multiplied by retry count.
	const maxScanWorkers = 64
	workerCount := min(maxScanWorkers, len(scanConfig.Endpoints))
	jobs := make(chan int)
	results := make(chan ScanResult, len(scanConfig.Endpoints))

	var workers sync.WaitGroup
	workers.Add(workerCount)
	for workerID := 0; workerID < workerCount; workerID++ {
		go func() {
			defer workers.Done()
			for index := range jobs {
				endpoint := scanConfig.Endpoints[index]
				if scanConfig.EndpointStaggeringMs > 0 {
					time.Sleep(time.Duration(scanConfig.EndpointStaggeringMs) * time.Millisecond)
				}

				proxyURL := &url.URL{
					Scheme: "http",
					Host:   fmt.Sprintf("127.0.0.1:%d", 1080+index),
				}
				transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
				client := &http.Client{
					Timeout:   2 * time.Second,
					Transport: transport,
				}

				retries := scanConfig.IPv4Retries
				if strings.HasPrefix(endpoint, "[") {
					retries = scanConfig.IPv6Retries
				}
				if retries < 1 {
					retries = 1
				}

				var successCount int
				var totalLatency int64
				for attempt := 0; attempt < retries; attempt++ {
					if attempt > 0 && scanConfig.RetryStaggeringMs > 0 {
						time.Sleep(time.Duration(scanConfig.RetryStaggeringMs) * time.Millisecond)
					}
					started := time.Now()
					resp, requestErr := client.Head("http://www.gstatic.com/generate_204")
					latency := time.Since(started).Milliseconds()
					if requestErr == nil && resp != nil {
						if resp.StatusCode == http.StatusNoContent {
							successCount++
							totalLatency += latency
						}
						if resp.Body != nil {
							_ = resp.Body.Close()
						}
					}
				}
				transport.CloseIdleConnections()

				if successCount == 0 {
					log.Printf("[%d] %s -> %s\n", index+1, fmtStr(endpoint, ORANGE, false), fmtStr("Failed", RED, true))
					continue
				}

				avgLatency := totalLatency / int64(successCount)
				lossRate := float64(retries-successCount) / float64(retries) * 100
				results <- ScanResult{Endpoint: endpoint, Loss: lossRate, Latency: avgLatency}
				log.Printf("[%d] %s -> %s - %s %.1f %% - %s %d ms\n",
					index+1,
					fmtStr(endpoint, ORANGE, false),
					fmtStr("Success", GREEN, true),
					fmtStr("Loss rate:", "", true),
					lossRate,
					fmtStr("Avg. Latency:", "", true),
					avgLatency,
				)
			}
		}()
	}

	go func() {
		defer close(jobs)
		for index := range scanConfig.Endpoints {
			jobs <- index
		}
	}()
	go func() {
		workers.Wait()
		close(results)
	}()

	allResults := make([]ScanResult, 0, len(scanConfig.Endpoints))
	for result := range results {
		allResults = append(allResults, result)
	}

	return allResults, nil
}
