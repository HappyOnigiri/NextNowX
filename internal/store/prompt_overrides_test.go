package store_test

import (
	"context"
	"testing"

	"github.com/HappyOnigiri/nnx/internal/domain"
)

func TestPromptOverridesRoundTripAndPartialUpdates(t *testing.T) {
	ctx := context.Background()
	database, service := openTestService(t)
	project, err := service.CreateProject(ctx, "Prompt project", "")
	if err != nil {
		t.Fatal(err)
	}
	feature, err := service.CreateFeature(ctx, "Prompt feature", "", project.ID)
	if err != nil {
		t.Fatal(err)
	}
	design := "Project design {{task_id}}"
	batch := "Feature batch {{task_list}}"
	project, err = service.UpdateProject(ctx, project.ID, domain.ProjectUpdate{
		PromptOverrides: &domain.PromptTemplateOverridesUpdate{Design: &design},
	})
	if err != nil {
		t.Fatal(err)
	}
	feature, err = service.UpdateFeature(ctx, feature.ID, domain.FeatureUpdate{
		PromptOverrides: &domain.PromptTemplateOverridesUpdate{Batch: &batch},
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.PromptOverrides.Design != design ||
		project.PromptOverrides.Implementation != "" ||
		feature.PromptOverrides.Batch != batch {
		t.Fatalf("project=%+v feature=%+v", project.PromptOverrides, feature.PromptOverrides)
	}
	cleared := ""
	project, err = service.UpdateProject(ctx, project.ID, domain.ProjectUpdate{
		PromptOverrides: &domain.PromptTemplateOverridesUpdate{Design: &cleared},
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.PromptOverrides.Design != "" {
		t.Fatalf("design override=%q, want cleared", project.PromptOverrides.Design)
	}
	// feature を移しても固有の上書きは残り、継承する種類だけ移動先の project に従う。
	second, err := service.CreateProject(ctx, "Second project", "")
	if err != nil {
		t.Fatal(err)
	}
	feature, err = service.UpdateFeature(ctx, feature.ID, domain.FeatureUpdate{ProjectID: &second.ID})
	if err != nil {
		t.Fatal(err)
	}
	if feature.ProjectID != second.ID || feature.PromptOverrides.Batch != batch {
		t.Fatalf("moved feature=%+v", feature)
	}
	_ = database
}

func TestPromptOverrideValidationAndArchiveGuard(t *testing.T) {
	ctx := context.Background()
	_, service := openTestService(t)
	project, err := service.CreateProject(ctx, "Prompt project", "")
	if err != nil {
		t.Fatal(err)
	}
	feature, err := service.CreateFeature(ctx, "Prompt feature", "", project.ID)
	if err != nil {
		t.Fatal(err)
	}
	bad := "no required placeholder"
	_, err = service.UpdateProject(ctx, project.ID, domain.ProjectUpdate{
		PromptOverrides: &domain.PromptTemplateOverridesUpdate{Design: &bad},
	})
	if domain.ErrorCode(err) != domain.DomainErrorCodeInvalidPromptTemplate {
		t.Fatalf("error=%v code=%s", err, domain.ErrorCode(err))
	}
	archived := true
	if _, err := service.UpdateProject(ctx, project.ID, domain.ProjectUpdate{Archived: &archived}); err != nil {
		t.Fatal(err)
	}
	valid := "Feature design {{task_id}}"
	_, err = service.UpdateFeature(ctx, feature.ID, domain.FeatureUpdate{
		PromptOverrides: &domain.PromptTemplateOverridesUpdate{Design: &valid},
	})
	if domain.ErrorCode(err) != domain.DomainErrorCodeArchivedReadOnly {
		t.Fatalf("archived feature error=%v code=%s", err, domain.ErrorCode(err))
	}
}
