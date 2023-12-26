package ui

import (
	"context"

	"github.com/henderiw/git-loader/pkg/auth"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
)

type ApiserverUserInfoProvider struct{}

var _ auth.UserInfoProvider = &ApiserverUserInfoProvider{}

func (p *ApiserverUserInfoProvider) GetUserInfo(ctx context.Context) *auth.UserInfo {
	userinfo, ok := request.UserFrom(ctx)
	if !ok {
		return nil
	}

	name := userinfo.GetName()
	if name == "" {
		return nil
	}

	for _, group := range userinfo.GetGroups() {
		if group == user.AllAuthenticated {
			return &auth.UserInfo{
				Name:  name, // k8s authentication only provides single name; use it for both values for now.
				Email: name,
			}
		}
	}

	return nil
}
