package store

import (
	"context"
	"database/sql"
	"errors"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/HappyOnigiri/NextNowX/internal/db"
	"github.com/HappyOnigiri/NextNowX/internal/domain"
)

func (s *Store) CreateTaskGroup(ctx context.Context, featureID, name string) (domain.TaskGroup, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.TaskGroup{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := db.New(tx)
	feature, err := q.GetFeatureByPublicID(ctx, featureID)
	if err != nil {
		return domain.TaskGroup{}, mapNotFound(err, "feature", featureID)
	}
	id, err := nextPublicID(ctx, q, "task_group", "G-")
	if err != nil {
		return domain.TaskGroup{}, err
	}
	group, err := q.CreateTaskGroup(ctx, db.CreateTaskGroupParams{
		PublicID: id, FeatureID: feature.ID, Name: name,
	})
	if err != nil {
		return domain.TaskGroup{}, mapTaskGroupNameError(err, name)
	}
	if err := tx.Commit(); err != nil {
		return domain.TaskGroup{}, err
	}
	return domainTaskGroup(group, feature.PublicID), nil
}

func (s *Store) GetTaskGroup(ctx context.Context, id string) (domain.TaskGroup, error) {
	q := db.New(s.db)
	group, err := q.GetTaskGroup(ctx, id)
	if err != nil {
		return domain.TaskGroup{}, mapNotFound(err, "task group", id)
	}
	return domainTaskGroup(group, publicFeatureID(ctx, q, group.FeatureID)), nil
}

func (s *Store) UpdateTaskGroup(
	ctx context.Context,
	group domain.TaskGroup,
	name, color *string,
) (domain.TaskGroup, error) {
	params := db.UpdateTaskGroupParams{PublicID: group.ID}
	if name != nil {
		params.Name = sql.NullString{String: *name, Valid: true}
		group.Name = *name
	}
	if color != nil {
		params.Color = sql.NullString{String: *color, Valid: true}
	}
	row, err := db.New(s.db).UpdateTaskGroup(ctx, params)
	if err != nil {
		return domain.TaskGroup{}, mapTaskGroupNameError(mapNotFound(err, "task group", group.ID), group.Name)
	}
	return domainTaskGroup(row, group.FeatureID), nil
}

func (s *Store) DeleteTaskGroup(ctx context.Context, id string, cascade bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := db.New(tx)
	if !cascade {
		hasDocuments, err := q.TaskGroupHasDocuments(ctx, sql.NullString{String: id, Valid: true})
		if err != nil {
			return err
		}
		if hasDocuments {
			return domain.NewError(domain.DomainErrorCodeReferencesExist, "task group has documents; pass --cascade")
		}
	}
	count, err := q.DeleteTaskGroup(ctx, id)
	if err != nil {
		return err
	}
	if count == 0 {
		return mapNotFound(sql.ErrNoRows, "task group", id)
	}
	return tx.Commit()
}

func validateTaskGroupAssignment(ctx context.Context, q *db.Queries, featureID, groupID string) error {
	if groupID == "" {
		return nil
	}
	group, err := q.GetTaskGroup(ctx, groupID)
	if err != nil {
		return mapNotFound(err, "task group", groupID)
	}
	if group.FeatureID != featureID {
		return domain.NewError(domain.DomainErrorCodeInvalidParent, "task group %q belongs to another feature", groupID)
	}
	return nil
}

func domainTaskGroup(row db.TaskGroup, featureID string) domain.TaskGroup {
	return domain.TaskGroup{ID: row.PublicID, FeatureID: featureID, Name: row.Name, Color: row.Color}
}

func mapTaskGroupNameError(err error, name string) error {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		return domain.NewError(
			domain.DomainErrorCodeDuplicateTaskGroupName,
			"task group name %q already exists in this feature",
			name,
		)
	}
	return err
}
