package dialog_grpc

import (
	"context"

	dialogv1 "github.com/kirban/social-media/api/proto/gen/dialog/v1"
	"github.com/kirban/social-media/internal/model"
	"github.com/kirban/social-media/internal/service"
)

// DialogGRPCClient implements DialogServiceInterface by calling the
// extracted dialog-service over gRPC. It is the gateway-side half of the
// Strangler Fig split: internal/api's handlers and api.gen.go are unaware
// this is not the in-process DialogService — they only see the interface.
type DialogGRPCClient struct {
	client dialogv1.DialogServiceClient
}

// NewDialogGRPCClient wraps an already-constructed dialogv1.DialogServiceClient.
// The caller dials the underlying connection (with the request-id-propagating
// client interceptor from internal/grpcinterceptor attached) and builds the
// client with dialogv1.NewDialogServiceClient.
func NewDialogGRPCClient(client dialogv1.DialogServiceClient) *DialogGRPCClient {
	return &DialogGRPCClient{client: client}
}

var _ service.DialogServiceInterface = (*DialogGRPCClient)(nil)

func (c *DialogGRPCClient) ListDialogs(ctx context.Context, srcUser model.UserID) ([]model.DialogSummary, error) {
	resp, err := c.client.ListDialogs(ctx, &dialogv1.ListDialogsRequest{SrcUser: srcUser})
	if err != nil {
		return nil, err
	}

	summaries := make([]model.DialogSummary, len(resp.GetDialogs()))
	for i, d := range resp.GetDialogs() {
		summaries[i] = model.DialogSummary{DialogID: d.GetDialogId(), UserID: d.GetUserId()}
	}
	return summaries, nil
}

func (c *DialogGRPCClient) GetMessages(ctx context.Context, srcUser, dstUser model.UserID, limit, offset int64) ([]model.DialogMessage, error) {
	resp, err := c.client.GetMessages(ctx, &dialogv1.GetMessagesRequest{SrcUser: srcUser, DstUser: dstUser, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}

	messages := make([]model.DialogMessage, len(resp.GetMessages()))
	for i, m := range resp.GetMessages() {
		messages[i] = model.DialogMessage{
			From:      m.GetFrom(),
			To:        m.GetTo(),
			DialogID:  m.GetDialogId(),
			Text:      m.GetText(),
			CreatedAt: m.GetCreatedAt().AsTime(),
			UpdatedAt: m.GetUpdatedAt().AsTime(),
		}
	}
	return messages, nil
}

func (c *DialogGRPCClient) SendMessage(ctx context.Context, srcUser, dstUser model.UserID, text string) (*model.DialogMessageID, error) {
	resp, err := c.client.SendMessage(ctx, &dialogv1.SendMessageRequest{SrcUser: srcUser, DstUser: dstUser, Text: text})
	if err != nil {
		return nil, err
	}
	id := resp.GetMessageId()
	return &id, nil
}
