// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the Apache v2.0 License.

package virtualmachine

import (
	"context"
	"fmt"

	"github.com/microsoft/moc-sdk-for-go/services"
	"github.com/microsoft/moc-sdk-for-go/services/compute"
	"github.com/microsoft/moc/pkg/auth"
	"github.com/microsoft/moc/pkg/config"
	"github.com/microsoft/moc/pkg/errors"
	"github.com/microsoft/moc/pkg/marshal"
	prototags "github.com/microsoft/moc/pkg/tags"
	wssdcloudproto "github.com/microsoft/moc/rpc/common"

	wssdcloudclient "github.com/microsoft/moc-sdk-for-go/pkg/client"
	wssdcloudcompute "github.com/microsoft/moc/rpc/cloudagent/compute"
)

type client struct {
	wssdcloudcompute.VirtualMachineAgentClient
}

// newVirtualMachineClient - creates a client session with the backend wssdcloud agent
func newVirtualMachineClient(subID string, authorizer auth.Authorizer) (*client, error) {
	c, err := wssdcloudclient.GetVirtualMachineClient(&subID, authorizer)
	if err != nil {
		return nil, err
	}
	return &client{c}, nil
}

// Get
func (c *client) Get(ctx context.Context, group, name string) (*[]compute.VirtualMachine, error) {
	request, err := c.getVirtualMachineRequest(wssdcloudproto.Operation_GET, group, name, nil)
	if err != nil {
		return nil, err
	}
	response, err := c.VirtualMachineAgentClient.Invoke(ctx, request)
	if err != nil {
		services.HandleGRPCError(err)

		return nil, err
	}
	return c.getVirtualMachineFromResponse(response, group), nil
}

// Get
func (c *client) get(ctx context.Context, group, name string) ([]*wssdcloudcompute.VirtualMachine, error) {
	request, err := c.getVirtualMachineRequest(wssdcloudproto.Operation_GET, group, name, nil)
	if err != nil {
		return nil, err
	}
	response, err := c.VirtualMachineAgentClient.Invoke(ctx, request)
	if err != nil {
		services.HandleGRPCError(err)

		return nil, err
	}
	return response.GetVirtualMachines(), nil
}

// CreateOrUpdate
func (c *client) CreateOrUpdate(ctx context.Context, group, name string, sg *compute.VirtualMachine) (*compute.VirtualMachine, error) {
	request, err := c.getVirtualMachineRequest(wssdcloudproto.Operation_POST, group, name, sg)
	if err != nil {
		return nil, err
	}
	response, err := c.VirtualMachineAgentClient.Invoke(ctx, request)
	if err != nil {
		services.HandleGRPCError(err)

		return nil, err
	}
	vms := c.getVirtualMachineFromResponse(response, group)
	if len(*vms) == 0 {
		return nil, fmt.Errorf("Creation of Virtual Machine failed to unknown reason.")
	}

	return &(*vms)[0], nil
}

// Delete methods invokes create or update on the client
func (c *client) Delete(ctx context.Context, group, name string) error {
	vm, err := c.Get(ctx, group, name)
	if err != nil {
		return err
	}
	if len(*vm) == 0 {
		return fmt.Errorf("Virtual Machine [%s] not found", name)
	}

	request, err := c.getVirtualMachineRequest(wssdcloudproto.Operation_DELETE, group, name, &(*vm)[0])
	if err != nil {
		return err
	}
	_, err = c.VirtualMachineAgentClient.Invoke(ctx, request)
	services.HandleGRPCError(err)
	return err
}

// Query
func (c *client) Query(ctx context.Context, group, query string) (*[]compute.VirtualMachine, error) {
	vms, err := c.Get(ctx, group, "")
	if err != nil {
		return nil, err
	}

	filteredBytes, err := config.MarshalOutput(*vms, query, "json")
	if err != nil {
		return nil, err
	}

	err = marshal.FromJSONBytes(filteredBytes, vms)
	if err != nil {
		return nil, err
	}

	return vms, nil
}

// Stop
func (c *client) Stop(ctx context.Context, group, name string) (err error) {
	request, err := c.getVirtualMachineOperationRequest(ctx, wssdcloudproto.VirtualMachineOperation_STOP, group, name)
	if err != nil {
		return
	}

	_, err = c.VirtualMachineAgentClient.Operate(ctx, request)
	return
}

// Start
func (c *client) Start(ctx context.Context, group, name string) (err error) {
	request, err := c.getVirtualMachineOperationRequest(ctx, wssdcloudproto.VirtualMachineOperation_START, group, name)
	if err != nil {
		return
	}

	_, err = c.VirtualMachineAgentClient.Operate(ctx, request)
	return
}

// RunCommand
func (c *client) RunCommand(ctx context.Context, group, name string, request *compute.VirtualMachineRunCommandRequest) (response *compute.VirtualMachineRunCommandResponse, err error) {
	mocRequest, err := c.getVirtualMachineRunCommandRequest(ctx, group, name, request)
	if err != nil {
		return
	}

	mocResponse, err := c.VirtualMachineAgentClient.RunCommand(ctx, mocRequest)
	if err != nil {
		return
	}
	response, err = c.getVirtualMachineRunCommandResponse(mocResponse)
	return
}

// Private methods
func (c *client) getVirtualMachineRunCommandRequest(ctx context.Context, group, name string, request *compute.VirtualMachineRunCommandRequest) (mocRequest *wssdcloudcompute.VirtualMachineRunCommandRequest, err error) {
	vms, err := c.get(ctx, group, name)
	if err != nil {
		return
	}

	if len(vms) != 1 {
		err = errors.Wrapf(errors.InvalidInput, "Multiple Virtual Machines found in group %s with name %s", group, name)
		return
	}
	vm := vms[0]

	var params []*wssdcloudproto.VirtualMachineRunCommandInputParameter
	if request.Parameters != nil {
		params = make([]*wssdcloudproto.VirtualMachineRunCommandInputParameter, len(*request.Parameters))
		for i, param := range *request.Parameters {
			tmp := &wssdcloudproto.VirtualMachineRunCommandInputParameter{
				Name:  *param.Name,
				Value: *param.Value,
			}
			params[i] = tmp
		}
	}

	var scriptSource wssdcloudproto.VirtualMachineRunCommandScriptSource
	if request.Source.Script != nil {
		scriptSource.Script = *request.Source.Script
	}
	if request.Source.ScriptURI != nil {
		scriptSource.ScriptURI = *request.Source.ScriptURI
	}
	if request.Source.CommandID != nil {
		scriptSource.CommandID = *request.Source.CommandID
	}

	mocRequest = &wssdcloudcompute.VirtualMachineRunCommandRequest{
		VirtualMachine:            vm,
		RunCommandInputParameters: params,
		Source:                    &scriptSource,
	}

	if request.RunAsUser != nil {
		mocRequest.RunAsUser = *request.RunAsUser
	}
	if request.RunAsPassword != nil {
		mocRequest.RunAsPassword = *request.RunAsPassword
	}
	return
}

func (c *client) getVirtualMachineRunCommandResponse(mocResponse *wssdcloudcompute.VirtualMachineRunCommandResponse) (*compute.VirtualMachineRunCommandResponse, error) {
	var executionState compute.ExecutionState
	switch mocResponse.GetInstanceView().ExecutionState {
	case wssdcloudproto.VirtualMachineRunCommandExecutionState_ExecutionState_UNKNOWN:
		executionState = compute.ExecutionStateUnknown
	case wssdcloudproto.VirtualMachineRunCommandExecutionState_ExecutionState_SUCCEEDED:
		executionState = compute.ExecutionStateSucceeded
	case wssdcloudproto.VirtualMachineRunCommandExecutionState_ExecutionState_FAILED:
		executionState = compute.ExecutionStateFailed
	default:
		return nil, errors.Wrapf(errors.NotSupported, "Unknown execution state reported for virtual machine run command")
	}

	instanceView := &compute.VirtualMachineRunCommandInstanceView{
		ExecutionState: executionState,
		ExitCode:       &mocResponse.GetInstanceView().ExitCode,
		Output:         &mocResponse.GetInstanceView().Output,
		Error:          &mocResponse.GetInstanceView().Error,
	}

	response := &compute.VirtualMachineRunCommandResponse{
		InstanceView: instanceView,
	}
	return response, nil
}

func (c *client) getVirtualMachineFromResponse(response *wssdcloudcompute.VirtualMachineResponse, group string) *[]compute.VirtualMachine {
	vms := []compute.VirtualMachine{}
	for _, vm := range response.GetVirtualMachines() {
		vms = append(vms, *(c.getVirtualMachine(vm, group)))
	}

	return &vms
}

func (c *client) getVirtualMachineRequest(opType wssdcloudproto.Operation, group, name string, vmss *compute.VirtualMachine) (*wssdcloudcompute.VirtualMachineRequest, error) {
	request := &wssdcloudcompute.VirtualMachineRequest{
		OperationType:   opType,
		VirtualMachines: []*wssdcloudcompute.VirtualMachine{},
	}
	var err error
	wssdvm := &wssdcloudcompute.VirtualMachine{
		Name:      name,
		GroupName: group,
	}
	if vmss != nil {
		wssdvm, err = c.getWssdVirtualMachine(vmss, group)
		if err != nil {
			return nil, err
		}
	}
	request.VirtualMachines = append(request.VirtualMachines, wssdvm)
	return request, nil
}

func (c *client) getVirtualMachineOperationRequest(ctx context.Context,
	opType wssdcloudproto.VirtualMachineOperation,
	group, name string) (request *wssdcloudcompute.VirtualMachineOperationRequest, err error) {

	vms, err := c.get(ctx, group, name)
	if err != nil {
		return
	}

	request = &wssdcloudcompute.VirtualMachineOperationRequest{
		OperationType:   opType,
		VirtualMachines: vms,
	}
	return
}

func getComputeTags(tags *wssdcloudproto.Tags) map[string]*string {
	return prototags.ProtoToMap(tags)
}

func getWssdTags(tags map[string]*string) *wssdcloudproto.Tags {
	return prototags.MapToProto(tags)
}
