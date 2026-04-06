package main

import (
	"C"
	"log"
	"sync"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
)

var clients sync.Map // stores all ZeroBusClient instances for cleanup in FLBPluginExit

//export FLBPluginRegister
func FLBPluginRegister(def unsafe.Pointer) int {
	return output.FLBPluginRegister(def, "zerobus", "Databricks ZeroBus output plugin")
}

//export FLBPluginInit
func FLBPluginInit(plugin unsafe.Pointer) int {
	cfg, err := parseConfig(plugin)
	if err != nil {
		log.Printf("[out_zerobus] configuration error: %v", err)
		return output.FLB_ERROR
	}

	client, err := NewZeroBusClient(cfg)
	if err != nil {
		log.Printf("[out_zerobus] initialization error: %v", err)
		return output.FLB_ERROR
	}

	output.FLBPluginSetContext(plugin, client)
	clients.Store(plugin, client)

	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	client, ok := output.FLBPluginGetContext(ctx).(*ZeroBusClient)
	if !ok || client == nil {
		log.Printf("[out_zerobus] failed to get plugin context")
		return output.FLB_ERROR
	}

	tagStr := C.GoString(tag)
	cfg := &client.FlushConfig

	decoder := output.NewDecoder(data, int(length))
	var records []interface{}
	var decodeErrors, convertErrors int

	for {
		ret, ts, record := output.GetRecord(decoder)
		if ret == -1 {
			// EOF or fatal decode error — no more records
			break
		}
		if ret != 0 {
			// Malformed record (ret -2 through -5); decoder has advanced past it
			decodeErrors++
			continue
		}

		jsonBytes, err := recordToJSON(ts, record, tagStr, cfg)
		if err != nil {
			convertErrors++
			log.Printf("[out_zerobus] failed to convert record: %v", err)
			continue
		}

		records = append(records, string(jsonBytes))
	}

	dropped := decodeErrors + convertErrors
	if dropped > 0 {
		log.Printf("[out_zerobus] skipped %d records (%d decode errors, %d convert errors)",
			dropped, decodeErrors, convertErrors)
	}

	if len(records) == 0 {
		if dropped > 0 {
			// All records in the chunk were malformed — signal error
			return output.FLB_ERROR
		}
		return output.FLB_OK
	}

	// Send the entire chunk as a single atomic batch
	if err := client.Ingest(records); err != nil {
		log.Printf("[out_zerobus] ingestion error: %v", err)
		if IsRetryable(err) {
			return output.FLB_RETRY
		}
		return output.FLB_ERROR
	}

	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {
	clients.Range(func(key, value interface{}) bool {
		if client, ok := value.(*ZeroBusClient); ok {
			if err := client.Close(); err != nil {
				log.Printf("[out_zerobus] error closing client: %v", err)
			}
		}
		clients.Delete(key)
		return true
	})
	return output.FLB_OK
}

func main() {}
