package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/notation-azure-kv/internal/version"
	"github.com/notaryproject/notation-go/plugin/proto"
)

func main() {
	if len(os.Args) < 2 {
		help()
		return
	}
	ctx := context.Background()
	var err error
	var resp any
	switch proto.Command(os.Args[1]) {
	case proto.CommandGetMetadata:
		resp = runGetMetadata()
	case proto.CommandDescribeKey:
		resp, err = runDescribeKey(ctx, os.Stdin)
	case proto.CommandGenerateSignature:
		resp, err = runSign(ctx, os.Stdin)
	default:
		err = fmt.Errorf("invalid command: %s", os.Args[1])
	}

	// output the response
	if err == nil {
		// ignore the error because the response only contains valid JSON field.
		jsonResp, _ := json.Marshal(resp)
		_, err = os.Stdout.Write(jsonResp)
	}

	// output the error
	if err != nil {
		data, _ := json.Marshal(wrapError(err))
		os.Stderr.Write(data)
		os.Exit(1)
	}
}

func wrapError(err error) *proto.RequestError {
	// already wrapped
	var nerr *proto.RequestError
	if errors.As(err, &nerr) {
		return nerr
	}

	// default error code
	code := proto.ErrorCodeGeneric

	// wrap Azure response
	var aerr *azcore.ResponseError
	if errors.As(err, &aerr) {
		switch aerr.StatusCode {
		case http.StatusUnauthorized:
			code = proto.ErrorCodeAccessDenied
		case http.StatusRequestTimeout:
			code = proto.ErrorCodeTimeout
		case http.StatusTooManyRequests:
			code = proto.ErrorCodeThrottled
		}
	}
	return &proto.RequestError{
		Code: code,
		Err:  err,
	}
}

func help() {
	fmt.Printf(`notation-azure-kv - Notation - Notary V2 Azure KV plugin

Usage:
  notation-azure-kv <command>

Version:
  %s

Commands:
  describe-key         Azure key description
  generate-signature   Sign artifacts with keys in Azure Key Vault
  get-plugin-metadata  Get plugin metadata

Documentation:
  https://github.com/notaryproject/notaryproject/blob/v1.0.0-rc.2/specs/plugin-extensibility.md#plugin-contract
`, version.GetVersion())
}
