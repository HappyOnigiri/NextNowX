package rpc

import (
	"context"

	"connectrpc.com/connect"

	nnxv1 "github.com/HappyOnigiri/NextNowX/gen/nnx/v1"
	"github.com/HappyOnigiri/NextNowX/internal/domain"
)

func (h *Handler) CreateTaskGroup(
	ctx context.Context,
	req *connect.Request[nnxv1.CreateTaskGroupRequest],
) (*connect.Response[nnxv1.CreateTaskGroupResponse], error) {
	group, err := h.service.CreateTaskGroup(ctx, req.Msg.GetFeatureId(), req.Msg.GetName())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&nnxv1.CreateTaskGroupResponse{TaskGroup: protoTaskGroup(group)}), nil
}

func (h *Handler) UpdateTaskGroup(
	ctx context.Context,
	req *connect.Request[nnxv1.UpdateTaskGroupRequest],
) (*connect.Response[nnxv1.UpdateTaskGroupResponse], error) {
	group, err := h.service.UpdateTaskGroup(ctx, req.Msg.GetId(), req.Msg.Name, req.Msg.Color)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&nnxv1.UpdateTaskGroupResponse{TaskGroup: protoTaskGroup(group)}), nil
}

func (h *Handler) DeleteTaskGroup(
	ctx context.Context,
	req *connect.Request[nnxv1.DeleteTaskGroupRequest],
) (*connect.Response[nnxv1.DeleteTaskGroupResponse], error) {
	if err := h.service.DeleteTaskGroup(ctx, req.Msg.GetId(), req.Msg.GetCascade()); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&nnxv1.DeleteTaskGroupResponse{}), nil
}

func protoTaskGroup(group domain.TaskGroup) *nnxv1.TaskGroup {
	return &nnxv1.TaskGroup{Id: group.ID, FeatureId: group.FeatureID, Name: group.Name, Color: group.Color}
}
