package grpcapi

import (
	"context"
	"time"

	"google.golang.org/grpc"

	"github.com/n1ckerr0r/shortener/internal/link"
)

type LinkService interface {
	Create(ctx context.Context, req link.CreateRequest) (*link.CreateResponse, error)
	Resolve(ctx context.Context, req link.ResolveRequest) (*link.ResolveResponse, error)
}

type Server struct {
	service LinkService
}

type LinkServiceServer interface {
	CreateLink(ctx context.Context, req *CreateLinkRequest) (*CreateLinkResponse, error)
	ResolveLink(ctx context.Context, req *ResolveLinkRequest) (*ResolveLinkResponse, error)
}

type CreateLinkRequest struct {
	URL       string     `json:"url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type CreateLinkResponse struct {
	ShortCode string `json:"short_code"`
}

type ResolveLinkRequest struct {
	Code       string `json:"code"`
	RemoteAddr string `json:"remote_addr,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
}

type ResolveLinkResponse struct {
	OriginalURL string `json:"original_url"`
}

func NewServer(service LinkService) *Server {
	return &Server{service: service}
}

func Register(server *grpc.Server, service LinkService) {
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: "shortener.link.v1.LinkService",
		HandlerType: (*LinkServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "CreateLink",
				Handler:    createLinkHandler,
			},
			{
				MethodName: "ResolveLink",
				Handler:    resolveLinkHandler,
			},
		},
	}, NewServer(service))
}

func (s *Server) CreateLink(ctx context.Context, req *CreateLinkRequest) (*CreateLinkResponse, error) {
	resp, err := s.service.Create(ctx, link.CreateRequest{
		OriginalURL: req.URL,
		ExpiresAt:   req.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	return &CreateLinkResponse{ShortCode: resp.ShortCode}, nil
}

func (s *Server) ResolveLink(ctx context.Context, req *ResolveLinkRequest) (*ResolveLinkResponse, error) {
	resp, err := s.service.Resolve(ctx, link.ResolveRequest{
		Code:       req.Code,
		RemoteAddr: req.RemoteAddr,
		UserAgent:  req.UserAgent,
	})
	if err != nil {
		return nil, err
	}

	return &ResolveLinkResponse{OriginalURL: resp.OriginalURL}, nil
}

func createLinkHandler(
	srv any,
	ctx context.Context,
	decode func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	req := new(CreateLinkRequest)
	if err := decode(req); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).CreateLink(ctx, req)
	}

	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/shortener.link.v1.LinkService/CreateLink",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).CreateLink(ctx, req.(*CreateLinkRequest))
	}

	return interceptor(ctx, req, info, handler)
}

func resolveLinkHandler(
	srv any,
	ctx context.Context,
	decode func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	req := new(ResolveLinkRequest)
	if err := decode(req); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*Server).ResolveLink(ctx, req)
	}

	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/shortener.link.v1.LinkService/ResolveLink",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*Server).ResolveLink(ctx, req.(*ResolveLinkRequest))
	}

	return interceptor(ctx, req, info, handler)
}
