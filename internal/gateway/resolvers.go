package gateway

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/azrsh/fragment-colocation-with-grpc/generated/backendv1"
	"github.com/azrsh/fragment-colocation-with-grpc/generated/backendv1/backendv1connect"
)

func NewResolvers(client *http.Client, userURL, organizationURL string) Resolvers {
	users := backendv1connect.NewUserServiceClient(client, userURL, connect.WithGRPC())
	organizations := backendv1connect.NewOrganizationServiceClient(client, organizationURL, connect.WithGRPC())
	return Resolvers{
		"Query.user": func(_ map[string]any, args map[string]any, ctx *ExecutionContext) (any, error) {
			value, err := scalar(args["id"], "ID", "Query.user(id:)", connect.CodeInvalidArgument)
			if err != nil {
				return nil, err
			}
			id := value.(string)
			return ctx.Once("user:"+id, func() (any, error) {
				requestCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				defer cancel()
				response, err := users.GetUser(requestCtx, connect.NewRequest(&backendv1.GetUserRequest{Id: id}))
				if err != nil {
					return nil, err
				}
				u := response.Msg.User
				if u == nil {
					return nil, nil
				}
				return map[string]any{"id": u.Id, "name": u.Name, "email": u.Email, "avatarUrl": u.AvatarUrl, "organizationId": u.OrganizationId}, nil
			})
		},
		"User.organization": func(source, args map[string]any, ctx *ExecutionContext) (any, error) {
			id, _ := source["organizationId"].(string)
			if id == "" {
				return nil, nil
			}
			return ctx.Once("organization:"+id, func() (any, error) {
				requestCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				defer cancel()
				response, err := organizations.GetOrganization(requestCtx, connect.NewRequest(&backendv1.GetOrganizationRequest{Id: id}))
				if err != nil {
					return nil, err
				}
				o := response.Msg.Organization
				if o == nil {
					return nil, nil
				}
				return map[string]any{"id": o.Id, "name": o.Name, "website": o.Website}, nil
			})
		},
	}
}
