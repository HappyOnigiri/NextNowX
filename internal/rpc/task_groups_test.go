package rpc_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	nnxv1 "github.com/HappyOnigiri/NextNowX/gen/nnx/v1"
)

func TestRPCTaskGroups(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	project := newRPCProject(t, ctx, client, "Groups")
	createFeature := func(title string) string {
		t.Helper()
		feature, err := client.CreateFeature(ctx, connect.NewRequest(&nnxv1.CreateFeatureRequest{
			Title: title, ProjectId: project,
		}))
		if err != nil {
			t.Fatal(err)
		}
		return feature.Msg.GetFeature().GetId()
	}
	feature, otherFeature := createFeature("Feature"), createFeature("Other feature")
	createGroup := func(featureID, name string, code connect.Code) string {
		t.Helper()
		group, err := client.CreateTaskGroup(ctx, connect.NewRequest(&nnxv1.CreateTaskGroupRequest{
			FeatureId: featureID, Name: name,
		}))
		if codeOfTaskGroupError(err) != code {
			t.Fatalf("create group %q err=%v, want code %v", name, err, code)
		}
		if err != nil {
			return ""
		}
		return group.Msg.GetTaskGroup().GetId()
	}
	createGroup(feature, "　 ", connect.CodeInvalidArgument)
	group := createGroup(feature, "  Checkout  ", 0)
	createGroup(feature, "Checkout", connect.CodeFailedPrecondition)
	otherGroup := createGroup(otherFeature, "Checkout", 0)
	secondGroup := createGroup(feature, "Delivery", 0)
	for _, invalid := range []struct {
		id   string
		code connect.Code
	}{{otherGroup, connect.CodeInvalidArgument}, {"G-999", connect.CodeNotFound}} {
		_, err := client.CreateTask(ctx, connect.NewRequest(&nnxv1.CreateTaskRequest{
			FeatureId: feature, Title: "Invalid assignment", GroupId: invalid.id,
		}))
		if connect.CodeOf(err) != invalid.code {
			t.Fatalf("create task in %q err=%v", invalid.id, err)
		}
	}
	task, err := client.CreateTask(ctx, connect.NewRequest(&nnxv1.CreateTaskRequest{
		FeatureId: feature, Title: "Assigned", GroupId: group,
	}))
	if err != nil || task.Msg.GetTask().GetGroupId() != group {
		t.Fatalf("task=%v err=%v", task, err)
	}
	taskID := task.Msg.GetTask().GetId()
	updateGroup := func(value *string, want string, code connect.Code) {
		t.Helper()
		result, err := client.UpdateTask(ctx, connect.NewRequest(&nnxv1.UpdateTaskRequest{
			Id: taskID, GroupId: value,
		}))
		if codeOfTaskGroupError(err) != code || (err == nil && result.Msg.GetTask().GetGroupId() != want) {
			t.Fatalf("update group task=%v err=%v", result, err)
		}
	}
	missing, empty := "G-999", ""
	updateGroup(&otherGroup, "", connect.CodeInvalidArgument)
	updateGroup(&missing, "", connect.CodeNotFound)
	updateGroup(nil, group, 0)
	updateGroup(&secondGroup, secondGroup, 0)
	updateGroup(&empty, "", 0)
	updateGroup(&group, group, 0)

	for _, rename := range []struct {
		name string
		code connect.Code
	}{{" ", connect.CodeInvalidArgument}, {"Delivery", connect.CodeFailedPrecondition}, {"  Launch  ", 0}} {
		result, err := client.RenameTaskGroup(ctx, connect.NewRequest(&nnxv1.RenameTaskGroupRequest{
			Id: group, Name: rename.name,
		}))
		if codeOfTaskGroupError(err) != rename.code ||
			(err == nil && (result.Msg.GetTaskGroup().GetName() != "Launch" || result.Msg.GetTaskGroup().GetId() != group)) {
			t.Fatalf("rename result=%v err=%v", result, err)
		}
	}
	archived := true
	assertReadOnly := func() {
		t.Helper()
		createGroup(feature, "Forbidden", connect.CodeFailedPrecondition)
		updateGroup(&empty, "", connect.CodeFailedPrecondition)
		_, err := client.RenameTaskGroup(ctx, connect.NewRequest(&nnxv1.RenameTaskGroupRequest{
			Id: group, Name: "Forbidden",
		}))
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Fatalf("archived rename error=%v", err)
		}
		_, err = client.DeleteTaskGroup(ctx, connect.NewRequest(&nnxv1.DeleteTaskGroupRequest{Id: group}))
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Fatalf("archived delete error=%v", err)
		}
	}
	setFeatureArchived := func(flag bool) {
		t.Helper()
		if _, err := client.UpdateFeature(ctx, connect.NewRequest(&nnxv1.UpdateFeatureRequest{
			Id: feature, Archived: &flag,
		})); err != nil {
			t.Fatal(err)
		}
	}
	setFeatureArchived(true)
	assertReadOnly()
	setFeatureArchived(false)
	if _, err := client.UpdateProject(ctx, connect.NewRequest(&nnxv1.UpdateProjectRequest{
		Id: project, Archived: &archived,
	})); err != nil {
		t.Fatal(err)
	}
	assertReadOnly()
	archived = false
	if _, err := client.UpdateProject(ctx, connect.NewRequest(&nnxv1.UpdateProjectRequest{
		Id: project, Archived: &archived,
	})); err != nil {
		t.Fatal(err)
	}
	if _, err := client.DeleteTaskGroup(ctx, connect.NewRequest(&nnxv1.DeleteTaskGroupRequest{Id: group})); err != nil {
		t.Fatal(err)
	}
	snapshot, err := client.GetSnapshot(ctx, connect.NewRequest(&nnxv1.GetSnapshotRequest{}))
	if err != nil || len(snapshot.Msg.GetSnapshot().GetTaskGroups()) != 2 {
		t.Fatalf("snapshot=%v err=%v", snapshot, err)
	}
	tasks := snapshot.Msg.GetSnapshot().GetTasks()
	if len(tasks) != 1 || tasks[0].GetId() != taskID || tasks[0].GetGroupId() != "" || !tasks[0].GetReady() {
		t.Fatalf("tasks after group deletion=%v", tasks)
	}
}

func codeOfTaskGroupError(err error) connect.Code {
	if err == nil {
		return 0
	}
	return connect.CodeOf(err)
}

func TestRPCGroupDocuments(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	project := newRPCProject(t, ctx, client, "References")
	feature, err := client.CreateFeature(ctx, connect.NewRequest(&nnxv1.CreateFeatureRequest{
		Title: "Rollout", ProjectId: project,
	}))
	if err != nil {
		t.Fatal(err)
	}
	featureID := feature.Msg.GetFeature().GetId()
	group, err := client.CreateTaskGroup(ctx, connect.NewRequest(&nnxv1.CreateTaskGroupRequest{
		FeatureId: featureID, Name: "Checklist",
	}))
	if err != nil {
		t.Fatal(err)
	}
	groupID := group.Msg.GetTaskGroup().GetId()
	for _, input := range []struct {
		groupID, featureID string
		plan               bool
		want               connect.Code
	}{
		{"G-999", "", false, connect.CodeNotFound},
		{groupID, featureID, false, connect.CodeInvalidArgument},
		{groupID, "", true, connect.CodeInvalidArgument},
	} {
		_, err := client.AddDocument(ctx, connect.NewRequest(&nnxv1.AddDocumentRequest{
			GroupId: input.groupID, FeatureId: input.featureID, IsImplementationPlan: input.plan,
			Source: &nnxv1.AddDocumentRequest_Markdown{Markdown: "# Checklist"},
		}))
		if connect.CodeOf(err) != input.want {
			t.Fatalf("add document error=%v, want %v", err, input.want)
		}
	}
	add := func() (*connect.Response[nnxv1.AddDocumentResponse], error) {
		t.Helper()
		return client.AddDocument(ctx, connect.NewRequest(&nnxv1.AddDocumentRequest{
			GroupId: groupID, Source: &nnxv1.AddDocumentRequest_Markdown{Markdown: "# Checklist"},
		}))
	}
	created, err := add()
	if err != nil || created.Msg.GetDocument().GetGroupId() != groupID {
		t.Fatalf("created=%v err=%v", created, err)
	}
	id := created.Msg.GetDocument().GetId()
	updated, err := client.UpdateDocument(ctx, connect.NewRequest(&nnxv1.UpdateDocumentRequest{
		Id: id, Title: new("Notes"),
	}))
	if err != nil || updated.Msg.GetDocument().GetGroupId() != groupID {
		t.Fatalf("updated=%v err=%v", updated, err)
	}
	setArchived := func(value bool) {
		t.Helper()
		if _, err := client.UpdateFeature(ctx, connect.NewRequest(&nnxv1.UpdateFeatureRequest{
			Id: featureID, Archived: &value,
		})); err != nil {
			t.Fatal(err)
		}
	}
	setArchived(true)
	if _, err := add(); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("archived add err=%v", err)
	}
	if _, err := client.DeleteDocument(ctx, connect.NewRequest(&nnxv1.DeleteDocumentRequest{
		Id: id,
	})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("archived document delete err=%v", err)
	}
	read, err := client.GetDocument(ctx, connect.NewRequest(&nnxv1.GetDocumentRequest{Id: id}))
	if err != nil || read.Msg.GetDocument().GetGroupId() != groupID || read.Msg.GetContent() != "# Checklist" {
		t.Fatalf("read=%v err=%v", read, err)
	}
	snapshot, err := client.GetSnapshot(ctx, connect.NewRequest(&nnxv1.GetSnapshotRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	docs := snapshot.Msg.GetSnapshot().GetDocuments()
	if len(docs) != 1 || docs[0].GetId() != id || docs[0].GetGroupId() != groupID {
		t.Fatalf("snapshot documents=%v", docs)
	}
	setArchived(false)
	deleteGroup := connect.NewRequest(&nnxv1.DeleteTaskGroupRequest{Id: groupID})
	if _, err := client.DeleteTaskGroup(ctx, deleteGroup); err != nil {
		t.Fatal(err)
	}
	getDocument := connect.NewRequest(&nnxv1.GetDocumentRequest{Id: id})
	if _, err := client.GetDocument(ctx, getDocument); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("document after group deletion err=%v", err)
	}
}
