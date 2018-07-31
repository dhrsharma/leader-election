package leadership

import "golang.org/x/net/context"

type Server struct {
	CurrStatus string
	Address    string
	UniqueID   string
}

func (s *Server) Status(ctx context.Context, in *StatusRequest) (*StatusReply, error) {
	return &StatusReply{Message: s.CurrStatus, UniqueId: s.UniqueID}, nil
}
