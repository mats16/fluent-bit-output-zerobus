package main

import (
	"errors"
	"fmt"
	"log"

	zerobus "github.com/databricks/zerobus-sdk/go"
)

// FlushConfig holds the subset of PluginConfig needed during flush operations.
type FlushConfig struct {
	AddTag  bool
	TimeKey string
}

// ZeroBusClient wraps the ZeroBus SDK and provides a simplified interface
// for the Fluent Bit output plugin.
type ZeroBusClient struct {
	sdk         *zerobus.ZerobusSdk
	stream      *zerobus.ZerobusStream
	FlushConfig FlushConfig
}

// NewZeroBusClient creates a new ZeroBus client with the given configuration.
func NewZeroBusClient(cfg *PluginConfig) (*ZeroBusClient, error) {
	sdk, err := zerobus.NewZerobusSdk(cfg.ZerobusEndpoint, cfg.WorkspaceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create ZeroBus SDK: %w", err)
	}

	options := zerobus.DefaultStreamConfigurationOptions()
	options.RecordType = zerobus.RecordTypeJson

	stream, err := sdk.CreateStream(
		zerobus.TableProperties{TableName: cfg.TableName},
		cfg.ClientID,
		cfg.ClientSecret,
		options,
	)
	if err != nil {
		sdk.Free()
		return nil, fmt.Errorf("failed to create ZeroBus stream: %w", err)
	}

	log.Printf("[out_zerobus] connected to %s, table: %s", cfg.ZerobusEndpoint, cfg.TableName)

	return &ZeroBusClient{
		sdk:    sdk,
		stream: stream,
		FlushConfig: FlushConfig{
			AddTag:  cfg.AddTag,
			TimeKey: cfg.TimeKey,
		},
	}, nil
}

// Ingest sends JSON records to ZeroBus with all-or-nothing semantics and waits
// for server acknowledgment. Each element must be a JSON string.
func (c *ZeroBusClient) Ingest(records []interface{}) error {
	if len(records) == 0 {
		return nil
	}

	offset, err := c.stream.IngestRecordsOffset(records)
	if err != nil {
		return err
	}

	return c.stream.WaitForOffset(offset)
}

// IsRetryable checks whether the given error is a retryable ZeroBus error.
func IsRetryable(err error) bool {
	var zbErr *zerobus.ZerobusError
	if errors.As(err, &zbErr) {
		return zbErr.Retryable()
	}
	return false
}

// Close gracefully shuts down the ZeroBus client.
func (c *ZeroBusClient) Close() error {
	var streamErr error
	if c.stream != nil {
		streamErr = c.stream.Close()
		c.stream = nil
	}
	if c.sdk != nil {
		c.sdk.Free()
		c.sdk = nil
	}
	return streamErr
}
