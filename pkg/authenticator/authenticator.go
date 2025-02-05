package authenticator

import (
	"context"
	"errors"

	"go4.org/syncutil"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type Authenticator struct {
	*oidc.Provider
	oauth2.Config
}

type AuthenticatorsService struct {
	Google *Authenticator
}

var once syncutil.Once

func New() (authenticators *AuthenticatorsService, err error) {
	google, err := GetGoogleAuthenticator()
	if err != nil {
		return nil, err
	}

	authenticators = &AuthenticatorsService{
		Google: google,
	}

	return authenticators, err
}

func (a *Authenticator) VerifyIDToken(
	ctx context.Context,
	token *oauth2.Token,
) (*oidc.IDToken, error) {
	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		err := errors.New("no id_token field in oauth2 token")
		return nil, err
	}

	oidcConfig := &oidc.Config{
		ClientID: a.ClientID,
	}

	return a.Verifier(oidcConfig).Verify(ctx, idToken)
}
