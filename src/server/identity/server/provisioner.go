package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	det "github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/userv1"

	"github.com/pachyderm/pachyderm/v2/src/internal/errors"
	"github.com/pachyderm/pachyderm/v2/src/internal/middleware/logging/client"
)

type provisioner interface {
	findUser(ctx context.Context, name string) (*user, error)
	createUser(context.Context, *user) (*user, error)
	findGroup(ctx context.Context, name string) (*group, error)
	createGroup(context.Context, *group) (*group, error)
	setUserGroups(context.Context, *user, []*group) error
	close() error
}

type user struct {
	id         int32
	name       string
	prevGroups []*group
}

type group struct {
	id   int32
	name string
}

type determinedProvisioner struct {
	conn  *grpc.ClientConn
	dc    det.DeterminedClient
	token string
}

type determinedConfig struct {
	MasterURL string
	Username  string
	Password  string
	TLS       bool
}

type errNotFound struct{}

func (e errNotFound) Error() string {
	return "not found"
}

func withDetToken(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-user-token", fmt.Sprintf("Bearer %s", token))
}

func newDeterminedProvisioner(ctx context.Context, config determinedConfig) (provisioner, error) {
	tlsOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
	if config.TLS {
		tlsOpt = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		}))
	}
	determinedURL, err := url.Parse(config.MasterURL)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing determined url %q", config.MasterURL)
	}
	conn, err := grpc.DialContext(ctx, determinedURL.Host, tlsOpt, grpc.WithStreamInterceptor(client.LogStream), grpc.WithUnaryInterceptor(client.LogUnary))
	if err != nil {
		return nil, errors.Wrapf(err, "dialing determined at %q", determinedURL.Host)
	}
	dc := det.NewDeterminedClient(conn)
	tok, err := mintDeterminedToken(ctx, dc, config.Username, config.Password)
	if err != nil {
		return nil, err
	}
	d := &determinedProvisioner{
		conn:  conn,
		dc:    dc,
		token: tok,
	}
	return d, nil
}

func mintDeterminedToken(ctx context.Context, dc det.DeterminedClient, username, password string) (string, error) {
	loginResp, err := dc.Login(ctx, &det.LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return "", errors.Wrap(err, "login as determined user")
	}
	return loginResp.Token, nil
}

func (d *determinedProvisioner) findUser(ctx context.Context, name string) (*user, error) {
	ctx = withDetToken(ctx, d.token)
	u, err := d.dc.GetUserByUsername(ctx, &det.GetUserByUsernameRequest{Username: name})
	if err != nil {
		return nil, errors.Wrapf(err, "get determined user %q", name)
	}
	gs, err := d.dc.GetGroups(ctx, &det.GetGroupsRequest{UserId: u.User.GetId()})
	if err != nil {
		return nil, errors.Wrapf(err, "get groups for determined user %q", name)
	}
	var prevGrps []*group
	for _, g := range gs.Groups {
		prevGrps = append(prevGrps, &group{
			id:   g.Group.GetGroupId(),
			name: g.Group.GetName(),
		})
	}
	return &user{
		id:         u.User.GetId(),
		name:       u.GetUser().GetUsername(),
		prevGroups: prevGrps,
	}, nil
}

func (d *determinedProvisioner) createUser(ctx context.Context, usr *user) (*user, error) {
	ctx = withDetToken(ctx, d.token)
	u, err := d.dc.PostUser(ctx, &det.PostUserRequest{User: &userv1.User{
		Username:    usr.name,
		Active:      true,
		Remote:      true,
		Admin:       false,
		DisplayName: usr.name,
	}})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to provision determiend user %v", usr.name)
	}
	return &user{id: u.User.Id, name: u.User.Username}, nil
}

func (d *determinedProvisioner) findGroup(ctx context.Context, name string) (*group, error) {
	ctx = withDetToken(ctx, d.token)
	g, err := d.dc.GetGroups(ctx, &det.GetGroupsRequest{Name: name})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find determiend group %v", name)
	}
	if len(g.Groups) == 0 {
		return nil, &errNotFound{}
	}
	return &group{id: g.Groups[0].Group.GroupId, name: name}, nil
}

func (d *determinedProvisioner) createGroup(ctx context.Context, grp *group) (*group, error) {
	ctx = withDetToken(ctx, d.token)
	g, err := d.dc.CreateGroup(ctx, &det.CreateGroupRequest{Name: grp.name})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create determiend group %v", grp.name)
	}
	return &group{id: g.Group.GroupId, name: g.Group.Name}, nil
}

func (d *determinedProvisioner) setUserGroups(ctx context.Context, usr *user, groups []*group) error {
	ctx = withDetToken(ctx, d.token)
	gIds := make(map[int32]struct{})
	for _, g := range groups {
		gIds[g.id] = struct{}{}
		if _, err := d.dc.UpdateGroup(ctx, &det.UpdateGroupRequest{
			GroupId:  g.id,
			AddUsers: []int32{usr.id},
		}); err != nil {
			return errors.Wrapf(err, "set user group %q for user %q", g.name, usr.name)
		}
	}
	for _, g := range usr.prevGroups {
		if _, ok := gIds[g.id]; !ok {
			if _, err := d.dc.UpdateGroup(ctx, &det.UpdateGroupRequest{
				RemoveUsers: []int32{usr.id},
			}); err != nil {
				return errors.Wrapf(err, "remove determined user %q from group %q", usr.name, g)
			}
		}
	}
	return nil
}

func (d *determinedProvisioner) close() error {
	return errors.Wrap(d.conn.Close(), "close the determined client's grpc connection")
}
