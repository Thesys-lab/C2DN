package main

import (
	"github.com/valyala/fasthttp"
	"time"
)

func serverFromDRAMCache(ctx *fasthttp.RequestCtx, startTs time.Time, remote bool, sz int) bool {
	// check frontend in-memory cache
	if feParam.RamCacheSize == 0 {
		return false
	}

	cachedContent, err := C2DNRamCache.Get(ctx.RequestURI())
	if err != nil {
		return false
	}

	ctx.Response.Header.Set("Via", "[cR]")
	if len(cachedContent) != sz {
		slogger.DPanicf("DRAM cached object size is different %d %d", len(cachedContent), sz)
	}

	ctx.SetBody(cachedContent[:sz])

	metricReqClient.WithLabelValues("feRAM").Inc()
	metricByteClient.WithLabelValues("feRAM").Add(float64(len(cachedContent)))

	//if LatChans.Started {
	//	if remote {
	//		LatChansRemote.RamCache <- float64(time.Since(startTs).Microseconds()) / 1000
	//	} else {
	//		LatChans.RamCache <- float64(time.Since(startTs).Microseconds()) / 1000
	//	}
	//}

	return true
}
