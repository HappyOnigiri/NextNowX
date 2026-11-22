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

func (s *Service) UpdateTaskGroup(ctx context.Context, id string, name, color *string) (domain.TaskGroup, error) {
	group, err := s.writableTaskGroup(ctx, id)
	if err != nil {
		return domain.TaskGroup{}, err
	}
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return domain.TaskGroup{}, domain.NewError(
				domain.DomainErrorCodeInvalidTitle,
				"task group name is required",
			)
		}
		name = &trimmed
	}
	if color != nil {
		normalized, err := domain.NormalizeHexColor(*color)
		if err != nil {
			return domain.TaskGroup{}, domain.NewError(domain.DomainErrorCodeInvalidTaskGroupColor, "%v", err)
		}
		color = &normalized
	}
	if name == nil && color == nil {
		return group, nil
	}
	return s.repository.UpdateTaskGroup(ctx, group, name, color)
}

func (s *Service) DeleteTaskGroup(ctx context.Context, id string, cascade bool) error {
	group, err := s.writableTaskGroup(ctx, id)
	if err != nil {
		return err
	}
	return s.repository.DeleteTaskGroup(ctx, group.ID, cascade)
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
