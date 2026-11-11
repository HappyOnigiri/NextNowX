package rpc_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	nnxv1 "github.com/HappyOnigiri/NextNowX/gen/nnx/v1"
)

func TestRPCTaskKinds(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	feature, err := client.CreateFeature(
		ctx,
		connect.NewRequest(
			&nnxv1.CreateFeatureRequest{Title: "Kinds", ProjectId: newRPCProject(t, ctx, client, "Kinds")},
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []nnxv1.TaskKind{
		nnxv1.TaskKind_TASK_KIND_UNSPECIFIED,
		nnxv1.TaskKind_TASK_KIND_DEVELOPMENT,
		nnxv1.TaskKind_TASK_KIND_WORK,
	} {
		task, err := client.CreateTask(
			ctx,
			connect.NewRequest(
				&nnxv1.CreateTaskRequest{FeatureId: feature.Msg.GetFeature().GetId(), Title: "Task", Kind: kind},
			),
		)
		if err != nil {
			t.Fatal(err)
		}
		want := kind
		if want == nnxv1.TaskKind_TASK_KIND_UNSPECIFIED {
			want = nnxv1.TaskKind_TASK_KIND_DEVELOPMENT
		}
		if task.Msg.GetTask().GetKind() != want {
			t.Fatalf("kind=%s want=%s", task.Msg.GetTask().GetKind(), want)
		}
		title := "Renamed"
		updated, err := client.UpdateTask(
			ctx,
			connect.NewRequest(&nnxv1.UpdateTaskRequest{Id: task.Msg.GetTask().GetId(), Title: &title}),
		)
		if err != nil || updated.Msg.GetTask().GetKind() != want {
			t.Fatalf("updated=%v err=%v", updated, err)
		}
		work := nnxv1.TaskKind_TASK_KIND_WORK
		updated, err = client.UpdateTask(
			ctx,
			connect.NewRequest(&nnxv1.UpdateTaskRequest{Id: task.Msg.GetTask().GetId(), Kind: &work}),
		)
		if err != nil || updated.Msg.GetTask().GetKind() != work {
			t.Fatalf("updated=%v err=%v", updated, err)
		}
		_, err = client.AttachPullRequest(
			ctx,
			connect.NewRequest(
				&nnxv1.AttachPullRequestRequest{
					TaskId: task.Msg.GetTask().GetId(),
					Url:    "https://github.com/acme/api/pull/1",
				},
			),
		)
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("attach err=%v", err)
		}
		for _, invalid := range []nnxv1.TaskKind{nnxv1.TaskKind_TASK_KIND_UNSPECIFIED, nnxv1.TaskKind(99)} {
			if _, err := client.UpdateTask(
				ctx,
				connect.NewRequest(&nnxv1.UpdateTaskRequest{Id: task.Msg.GetTask().GetId(), Kind: &invalid}),
			); connect.CodeOf(
				err,
			) != connect.CodeInvalidArgument {
				t.Fatalf("invalid kind err=%v", err)
			}
		}
	}
	if _, err := client.CreateTask(
		ctx,
		connect.NewRequest(
			&nnxv1.CreateTaskRequest{
				FeatureId: feature.Msg.GetFeature().GetId(),
				Title:     "Invalid",
				Kind:      nnxv1.TaskKind(99),
			},
		),
	); connect.CodeOf(
		err,
	) != connect.CodeInvalidArgument {
		t.Fatalf("create kind err=%v", err)
	}
}
