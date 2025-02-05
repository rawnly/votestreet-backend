package authenticator

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/voxelite-ai/env"
	"golang.org/x/oauth2"
)

// newAuthenticator creates a new authenticator for the specified issuer, client ID, client secret, redirect URI, and scopes.
// The function returns an error if the provider cannot be created.
func newAuthenticator(issuer, clientID, clientSecret, redirectURI string, scopes []string) (*Authenticator, error) {
	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		return nil, err
	}

	scopes = append(scopes, oidc.ScopeOpenID)

	authenticator := &Authenticator{
		Provider: provider,
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Endpoint:     provider.Endpoint(),
			Scopes:       scopes,
		},
	}

	return authenticator, nil
}

func GetGoogleAuthenticator() (*Authenticator, error) {
	return newAuthenticator(
		"https://accounts.google.com",
		env.String("GOOGLE_AUTH_CLIENT_ID"),
		env.String("GOOGLE_AUTH_CLIENT_SECRET"),
		env.String("GOOGLE_AUTH_REDIRECT_URI"),
		[]string{"profile", "email"},
	)
}
