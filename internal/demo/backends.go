package demo

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	"github.com/azrsh/fragment-colocation-with-grpc/generated/backendv1"
	"github.com/azrsh/fragment-colocation-with-grpc/generated/backendv1/backendv1connect"
	"github.com/azrsh/fragment-colocation-with-grpc/internal/network"
)

type fixtures struct{ onCall func(string, string) }

func (f fixtures) GetUser(ctx context.Context, request *connect.Request[backendv1.GetUserRequest]) (*connect.Response[backendv1.GetUserResponse], error) {
	f.onCall("UserService.GetUser", request.Msg.Id)
	users := []*backendv1.User{
		{Id: "u1", Name: "Aki Tanaka", Email: "aki@example.test", AvatarUrl: "/avatars/aki.svg", OrganizationId: "o1"},
		{Id: "u2", Name: "Ren Sato", Email: "ren@example.test", AvatarUrl: "/avatars/ren.svg", OrganizationId: "o2"},
		{Id: "u3", Name: "Mika Ito", Email: "mika@example.test", AvatarUrl: "/avatars/mika.svg"},
	}
	response := &backendv1.GetUserResponse{}
	for _, u := range users {
		if u.Id == request.Msg.Id {
			response.User = u
			break
		}
	}
	return connect.NewResponse(response), nil
}
func (f fixtures) GetOrganization(ctx context.Context, request *connect.Request[backendv1.GetOrganizationRequest]) (*connect.Response[backendv1.GetOrganizationResponse], error) {
	f.onCall("OrganizationService.GetOrganization", request.Msg.Id)
	organizations := []*backendv1.Organization{
		{Id: "o1", Name: "Northstar Studio", Website: "https://example.com/northstar"},
		{Id: "o2", Name: "Fieldwork Labs", Website: "https://example.com/fieldwork"},
	}
	response := &backendv1.GetOrganizationResponse{}
	for _, o := range organizations {
		if o.Id == request.Msg.Id {
			response.Organization = o
			break
		}
	}
	return connect.NewResponse(response), nil
}

type Backends struct{ User, Organization *network.Server }

func StartBackends(onCall func(string, string)) (*Backends, error) {
	if onCall == nil {
		onCall = func(string, string) {}
	}
	f := fixtures{onCall: onCall}
	userMux := http.NewServeMux()
	userMux.Handle(backendv1connect.NewUserServiceHandler(f))
	user, err := network.Serve(userMux, "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	orgMux := http.NewServeMux()
	orgMux.Handle(backendv1connect.NewOrganizationServiceHandler(f))
	organization, err := network.Serve(orgMux, "127.0.0.1:0")
	if err != nil {
		user.Close()
		return nil, err
	}
	return &Backends{User: user, Organization: organization}, nil
}
func (b *Backends) Close() error { return errors.Join(b.User.Close(), b.Organization.Close()) }
