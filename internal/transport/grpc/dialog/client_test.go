package dialog_grpc

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	dialogv1 "github.com/kirban/social-media/api/proto/gen/dialog/v1"
	"github.com/kirban/social-media/internal/model"
)

// fakeDialogServiceClient implements dialogv1.DialogServiceClient with
// canned responses, so DialogGRPCClient's field mapping can be tested
// without a real gRPC connection.
type fakeDialogServiceClient struct {
	listDialogsResp    *dialogv1.ListDialogsResponse
	getMessagesResp    *dialogv1.GetMessagesResponse
	sendMessageResp    *dialogv1.SendMessageResponse
	lastListReq        *dialogv1.ListDialogsRequest
	lastGetMessagesReq *dialogv1.GetMessagesRequest
	lastSendReq        *dialogv1.SendMessageRequest
}

func (f *fakeDialogServiceClient) ListDialogs(ctx context.Context, in *dialogv1.ListDialogsRequest, opts ...grpc.CallOption) (*dialogv1.ListDialogsResponse, error) {
	f.lastListReq = in
	return f.listDialogsResp, nil
}

func (f *fakeDialogServiceClient) GetMessages(ctx context.Context, in *dialogv1.GetMessagesRequest, opts ...grpc.CallOption) (*dialogv1.GetMessagesResponse, error) {
	f.lastGetMessagesReq = in
	return f.getMessagesResp, nil
}

func (f *fakeDialogServiceClient) SendMessage(ctx context.Context, in *dialogv1.SendMessageRequest, opts ...grpc.CallOption) (*dialogv1.SendMessageResponse, error) {
	f.lastSendReq = in
	return f.sendMessageResp, nil
}

func TestDialogGRPCClient_ListDialogs(t *testing.T) {
	fake := &fakeDialogServiceClient{
		listDialogsResp: &dialogv1.ListDialogsResponse{
			Dialogs: []*dialogv1.DialogSummary{
				{DialogId: "d1", UserId: "u2"},
			},
		},
	}
	c := NewDialogGRPCClient(fake)

	got, err := c.ListDialogs(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.lastListReq.GetSrcUser() != "u1" {
		t.Fatalf("SrcUser = %q, want %q", fake.lastListReq.GetSrcUser(), "u1")
	}

	want := []model.DialogSummary{{DialogID: "d1", UserID: "u2"}}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("ListDialogs() = %+v, want %+v", got, want)
	}
}

func TestDialogGRPCClient_GetMessages(t *testing.T) {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := created.Add(time.Minute)

	fake := &fakeDialogServiceClient{
		getMessagesResp: &dialogv1.GetMessagesResponse{
			Messages: []*dialogv1.DialogMessage{
				{
					From:      "u1",
					To:        "u2",
					DialogId:  "d1",
					Text:      "hi",
					CreatedAt: timestamppb.New(created),
					UpdatedAt: timestamppb.New(updated),
				},
			},
		},
	}
	c := NewDialogGRPCClient(fake)

	got, err := c.GetMessages(context.Background(), "u1", "u2", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.lastGetMessagesReq.GetLimit() != 10 || fake.lastGetMessagesReq.GetOffset() != 0 {
		t.Fatalf("unexpected request: %+v", fake.lastGetMessagesReq)
	}

	want := model.DialogMessage{From: "u1", To: "u2", DialogID: "d1", Text: "hi", CreatedAt: created, UpdatedAt: updated}
	if len(got) != 1 || !got[0].CreatedAt.Equal(want.CreatedAt) || !got[0].UpdatedAt.Equal(want.UpdatedAt) ||
		got[0].From != want.From || got[0].To != want.To || got[0].DialogID != want.DialogID || got[0].Text != want.Text {
		t.Fatalf("GetMessages() = %+v, want %+v", got, want)
	}
}

func TestDialogGRPCClient_SendMessage(t *testing.T) {
	fake := &fakeDialogServiceClient{
		sendMessageResp: &dialogv1.SendMessageResponse{MessageId: "m1"},
	}
	c := NewDialogGRPCClient(fake)

	id, err := c.SendMessage(context.Background(), "u1", "u2", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == nil || *id != "m1" {
		t.Fatalf("SendMessage() id = %v, want %q", id, "m1")
	}
	if fake.lastSendReq.GetText() != "hello" {
		t.Fatalf("Text = %q, want %q", fake.lastSendReq.GetText(), "hello")
	}
}
