package app

import (
	"context"
	"strings"

	"github.com/HappyOnigiri/NextNowX/internal/domain"
)

func (s *Service) CreateTaskGroup(ctx context.Context, featureID, name string) (domain.TaskGroup, error) {
	feature, err := s.ResolveFeature(ctx, featureID)
	if err != nil {
		return domain.TaskGroup{}, err
	}
	if err := s.guardFeature(ctx, feature); err != nil {
		return domain.TaskGroup{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.TaskGroup{}, domain.NewError(domain.DomainErrorCodeInvalidTitle, "task group name is required")
	}
	return s.repository.CreateTaskGroup(ctx, feature.ID, name)
}

func (s *Service) RenameTaskGroup(ctx context.Context, id, name string) (domain.TaskGroup, error) {
	group, err := s.writableTaskGroup(ctx, id)
	if err != nil {
		return domain.TaskGroup{}, err
	}
	group.Name = strings.TrimSpace(name)
	if group.Name == "" {
		return domain.TaskGroup{}, domain.NewError(domain.DomainErrorCodeInvalidTitle, "task group name is required")
	}
	return s.repository.RenameTaskGroup(ctx, group)
}

func (s *Service) DeleteTaskGroup(ctx context.Context, id string) error {
	group, err := s.writableTaskGroup(ctx, id)
	if err != nil {
		return err
	}
	return s.repository.DeleteTaskGroup(ctx, group.ID)
}

func (s *Service) writableTaskGroup(ctx context.Context, id string) (domain.TaskGroup, error) {
	group, err := s.repository.GetTaskGroup(ctx, id)
	if err != nil {
		return domain.TaskGroup{}, err
	}
	feature, err := s.ResolveFeature(ctx, group.FeatureID)
	if err != nil {
		return domain.TaskGroup{}, err
	}
	return group, s.guardFeature(ctx, feature)
}
