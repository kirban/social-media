// Package dialogserver adapts internal/service's DialogServiceInterface to
// the generated dialogv1.DialogServiceServer gRPC contract. All dialog
// business logic stays in internal/service/dialog.go, untouched — this is a
// pure translation layer between model.* and the proto types.
package dialogserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	dialogv1 "github.com/kirban/social-media/api/proto/gen/dialog/v1"
	"github.com/kirban/social-media/internal/service"
)

type Server struct {
	dialogv1.UnimplementedDialogServiceServer
	svc service.DialogServiceInterface
}

func New(svc service.DialogServiceInterface) *Server {
	return &Server{svc: svc}
}

func (s *Server) ListDialogs(ctx context.Context, req *dialogv1.ListDialogsRequest) (*dialogv1.ListDialogsResponse, error) {
	summaries, err := s.svc.ListDialogs(ctx, req.GetSrcUser())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &dialogv1.ListDialogsResponse{Dialogs: make([]*dialogv1.DialogSummary, len(summaries))}
	for i, d := range summaries {
		resp.Dialogs[i] = &dialogv1.DialogSummary{DialogId: d.DialogID, UserId: d.UserID}
	}
	return resp, nil
}

func (s *Server) GetMessages(ctx context.Context, req *dialogv1.GetMessagesRequest) (*dialogv1.GetMessagesResponse, error) {
	msgs, err := s.svc.GetMessages(ctx, req.GetSrcUser(), req.GetDstUser(), req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &dialogv1.GetMessagesResponse{Messages: make([]*dialogv1.DialogMessage, len(msgs))}
	for i, m := range msgs {
		resp.Messages[i] = &dialogv1.DialogMessage{
			From:      m.From,
			To:        m.To,
			DialogId:  m.DialogID,
			Text:      m.Text,
			CreatedAt: timestamppb.New(m.CreatedAt),
			UpdatedAt: timestamppb.New(m.UpdatedAt),
		}
	}
	return resp, nil
}

func (s *Server) SendMessage(ctx context.Context, req *dialogv1.SendMessageRequest) (*dialogv1.SendMessageResponse, error) {
	id, err := s.svc.SendMessage(ctx, req.GetSrcUser(), req.GetDstUser(), req.GetText())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &dialogv1.SendMessageResponse{MessageId: *id}, nil
}
