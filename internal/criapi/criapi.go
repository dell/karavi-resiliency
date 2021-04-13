/*
 * Copyright (c) 2021. Dell Inc., or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 */
package criapi

import (
	"context"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"time"
)

type Client struct {
	CRIConn              *grpc.ClientConn              // A grpc client connection to CRI
	RuntimeServiceClient v1alpha2.RuntimeServiceClient // A RuntimeService climent
}

var CRIClient Client

var CRIClientDialRetry = 30 * time.Second

func NewCRIClient(criSock string, clientOpts ...grpc.DialOption) (*Client, error) {
	var err error
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		CRIClient.CRIConn, err = grpc.DialContext(ctx, criSock, grpc.WithInsecure())
		if err != nil || CRIClient.CRIConn == nil {
			var errMsg string
			if err == nil {
				errMsg = "No error returned, but CRIClient.CRIConn is nil"
			} else {
				errMsg = err.Error()
			}
			log.Errorf("Waiting on connection to CRI socket: %s: %s", criSock, errMsg)
			time.Sleep(CRIClientDialRetry)
		} else {
			break
		}
	}
	log.Infof("Connected to CRI: %s", criSock)
	CRIClient.RuntimeServiceClient = v1alpha2.NewRuntimeServiceClient(CRIClient.CRIConn)
	return &CRIClient, nil
}

func (cri *Client) Connected() bool {
	return cri.CRIConn != nil
}

func (cri *Client) Close() error {
	if cri.Connected() {
		if err := cri.CRIConn.Close(); err != nil {
			return err
		}
		cri.CRIConn = nil
		return nil
	}
	return nil
}

func (cri *Client) ListContainers(ctx context.Context, req *v1alpha2.ListContainersRequest) (*v1alpha2.ListContainersResponse, error) {
	return CRIClient.RuntimeServiceClient.ListContainers(ctx, req)
}

func (cri *Client) GetContainerInfo(ctx context.Context) (map[string]*ContainerInfo, error) {
	result := make(map[string]*ContainerInfo)
	client, err := NewCRIClient("unix:/var/run/dockershim.sock")
	if err != nil {
		return result, err
	}
	req := &v1alpha2.ListContainersRequest{}
	rep, err := client.ListContainers(context.Background(), req)
	if err != nil {
		return result, err
	}
	for _, cont := range rep.Containers {
		info := &ContainerInfo{
			ID:    cont.Id,
			Name:  cont.Metadata.Name,
			State: cont.State,
		}
		result[cont.Id] = info
	}
	err = client.Close()
	if err != nil {
		log.Infof("close error: %s", err)
	}
	return result, nil
}
