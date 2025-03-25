package wltbase

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"time"
)

func (e *env) CacheGet(ctx context.Context, u string, timeout, refresh time.Duration) ([]byte, error) {
	cacheKey := sha256.Sum256([]byte(u))

	// check if in cache
	cachebuf, err := e.DBSimpleGet([]byte("http_cache"), cacheKey[:])
	if err == nil {
		// found, return it
		cacheTime := time.Unix(int64(binary.BigEndian.Uint64(cachebuf[:8])), 0)
		if time.Since(cacheTime) <= refresh {
			// still fresh enough
			return cachebuf[8:], nil
		}
	}

	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		if cachebuf != nil {
			return cachebuf, nil
		}
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if cachebuf != nil {
			return cachebuf, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		if cachebuf != nil {
			return cachebuf, nil
		}
		return nil, err
	}

	if resp.StatusCode >= 300 {
		if cachebuf != nil {
			return cachebuf, nil
		}
		if len(buf) > 512 {
			buf = buf[:512]
		}
		return nil, fmt.Errorf("HTTP status %s on GET: %s", resp.Status, buf)
	}

	// current timestamp
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(time.Now().Unix()))

	// save in cache (ignore errors)
	e.DBSimpleSet([]byte("http_cache"), cacheKey[:], append(ts, buf...))

	return buf, nil
}
