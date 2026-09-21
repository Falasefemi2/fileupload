package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

type GoogleAuth struct {
	Config *oauth2.Config
}

func NewGoogleAuth(
	clientID string,
	clientSecret string,
	redirectURL string,
) *GoogleAuth {
	return &GoogleAuth{
		Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     google.Endpoint,
			Scopes: []string{
				"openid",
				"email",
				"profile",
			},
		},
	}
}

func (g *GoogleAuth) LoginURL(state string) string {
	return g.Config.AuthCodeURL(state)
}

func (g *GoogleAuth) ExchangeCode(
	ctx context.Context,
	code string,
) (*oauth2.Token, error) {
	return g.Config.Exchange(ctx, code)
}

func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func FetchGoogleUser(
	ctx context.Context,
	cfg *oauth2.Config,
	code string,
) (*GoogleUser, error) {
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	return GetGoogleUser(ctx, token)
}

func GetGoogleUser(
	ctx context.Context,
	token *oauth2.Token,
) (*GoogleUser, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	resp, err := client.Get(
		"https://www.googleapis.com/oauth2/v2/userinfo",
	)
	if err != nil {
		return nil, fmt.Errorf("get google user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"google userinfo returned status %d",
			resp.StatusCode,
		)
	}

	var user GoogleUser

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode google user: %w", err)
	}

	if user.ID == "" || user.Email == "" {
		return nil, fmt.Errorf("google returned incomplete user information")
	}

	return &user, nil
}
