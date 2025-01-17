// Copyright 2019, Oath Inc.
// Licensed under the terms of the Apache License 2.0. Please see LICENSE file in project root for terms.

package api

import (
	"context"
	"crypto"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/yahoo/crypki/config"
	"github.com/yahoo/crypki/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetBlobAvailableSigningKeys returns all available keys that can sign
func (s *SigningService) GetBlobAvailableSigningKeys(ctx context.Context, e *empty.Empty) (*proto.KeyMetas, error) {
	const methodName = "GetBlobAvailableSigningKeys"
	statusCode := http.StatusOK
	start := time.Now()
	var err error

	defer func() {
		log.Printf(`m=%s,st=%d,et=%d,err="%v"`, methodName, statusCode, timeElapsedSince(start), err)
	}()
	defer recoverIfPanicked(methodName)

	var keys []*proto.KeyMeta
	for id := range s.KeyUsages[config.BlobEndpoint] {
		keys = append(keys, &proto.KeyMeta{Identifier: id})
	}
	return &proto.KeyMetas{Keys: keys}, nil
}

// GetBlobSigningKey returns the public signing key of the
// specified key that signs the user's data.
func (s *SigningService) GetBlobSigningKey(ctx context.Context, keyMeta *proto.KeyMeta) (*proto.PublicKey, error) {
	const methodName = "GetBlobSigningKey"
	statusCode := http.StatusOK
	start := time.Now()
	var err error

	defer func() {
		log.Printf(`m=%s,st=%d,et=%d,err="%v"`, methodName, statusCode, timeElapsedSince(start), err)
	}()
	defer recoverIfPanicked(methodName)

	if keyMeta == nil {
		statusCode = http.StatusBadRequest
		err = fmt.Errorf("keyMeta is empty for %q", config.BlobEndpoint)
		return nil, status.Errorf(codes.InvalidArgument, "Bad request: %v", err)
	}

	if !s.KeyUsages[config.BlobEndpoint][keyMeta.Identifier] {
		statusCode = http.StatusBadRequest
		err = fmt.Errorf("cannot use key %q for %q", keyMeta.Identifier, config.BlobEndpoint)
		return nil, status.Errorf(codes.InvalidArgument, "Bad request: %v", err)
	}

	key, err := s.GetBlobSigningPublicKey(keyMeta.Identifier)
	if err != nil {
		statusCode = http.StatusInternalServerError
		return nil, status.Error(codes.Internal, "Internal server error")
	}
	return &proto.PublicKey{Key: string(key)}, nil
}

// PostSignBlob signs the digest using the specified key.
func (s *SigningService) PostSignBlob(ctx context.Context, request *proto.BlobSigningRequest) (*proto.Signature, error) {
	const methodName = "PostSignBlob"
	statusCode := http.StatusCreated
	start := time.Now()
	var err error

	defer func() {
		log.Printf(`m=%s,digest=%q,hash=%q,st=%d,et=%d,err="%v"`, methodName, request.GetDigest(), request.HashAlgorithm.String(), statusCode, timeElapsedSince(start), err)
	}()
	defer recoverIfPanicked(methodName)

	if request.KeyMeta == nil {
		statusCode = http.StatusBadRequest
		err = fmt.Errorf("request.keyMeta is empty for %q", config.BlobEndpoint)
		return nil, status.Errorf(codes.InvalidArgument, "Bad request: %v", err)
	}

	if !s.KeyUsages[config.BlobEndpoint][request.KeyMeta.Identifier] {
		statusCode = http.StatusBadRequest
		err = fmt.Errorf("cannot use key %q for %q", request.KeyMeta.Identifier, config.BlobEndpoint)
		return nil, status.Errorf(codes.InvalidArgument, "Bad request: %v", err)
	}

	digest, err := base64.StdEncoding.DecodeString(request.GetDigest())
	if err != nil {
		statusCode = http.StatusBadRequest
		return nil, status.Errorf(codes.InvalidArgument, "Bad request: %v", err)
	}

	signerOpts := getSignerOpts(request.HashAlgorithm.String())
	signature, err := s.Sign(digest, signerOpts, request.KeyMeta.Identifier)
	if err != nil {
		statusCode = http.StatusInternalServerError
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	base64Signature := base64.StdEncoding.EncodeToString(signature)
	return &proto.Signature{Signature: base64Signature}, nil
}

func getSignerOpts(hashAlgo string) crypto.SignerOpts {
	switch hashAlgo {
	case "SHA224":
		return crypto.SHA224
	case "SHA256":
		return crypto.SHA256
	case "SHA384":
		return crypto.SHA384
	case "SHA512":
		return crypto.SHA512
	default:
		return crypto.SHA512
	}
}
