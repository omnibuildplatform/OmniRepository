package application

import (
	"context"

	"github.com/omnibuildplatform/OmniRepository/app"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type RepoMonitor struct {
	app.UnimplementedRepoServerServer
}

func (s *RepoMonitor) CallLoadFrom(ctx context.Context, in *app.RepRequest) (*app.RepResponse, error) {

	return nil, status.Errorf(codes.Unimplemented, "method CallLoadFrom not implemented")

}
