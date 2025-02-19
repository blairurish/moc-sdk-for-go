// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license

package internal

import (
	"context"

	mocclient "github.com/microsoft/moc-sdk-for-go/pkg/client"
	"github.com/microsoft/moc-sdk-for-go/services"
	"github.com/microsoft/moc/pkg/auth"
	mocadmin "github.com/microsoft/moc/rpc/common/admin"
)

type client struct {
	mocadmin.DebugAgentClient
}

// NewDebugClient - creates a client session with the backend moc agent
func NewDebugClient(subID string, authorizer auth.Authorizer) (*client, error) {
	c, err := mocclient.GetDebugClient(&subID, authorizer)
	if err != nil {
		return nil, err
	}
	return &client{c}, nil
}

// Stacktrace
func (c *client) Stacktrace(ctx context.Context) (string, error) {
	request := getDebugRequest(mocadmin.DebugOperation_STACKTRACE)
	response, err := c.DebugAgentClient.Invoke(ctx, request)
	if err != nil {
		services.HandleGRPCError(err)
		return "", err
	}
	return response.Result, nil
}

func getDebugRequest(operation mocadmin.DebugOperation) *mocadmin.DebugRequest {
	return &mocadmin.DebugRequest{OperationType: operation}
}
