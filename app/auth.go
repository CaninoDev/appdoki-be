package app

import (
	"appdoki-be/app/repositories"
	"appdoki-be/config"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"net/http"
	"strings"
	"time"
)

// AuthHandler holds handler dependencies
type AuthHandler struct {
	appConfig config.AppConfig
	userRepo  repositories.UsersRepositoryInterface
	notifier  notifier
}

type AuthCodePayload struct {
	Code string `json:"code"`
}

type TokenPayload struct {
	Token string `json:"token"`
}

// NewOAuthHandler returns an initialized users handler with the required dependencies
func NewAuthHandler(
	appConfig config.AppConfig,
	userRepo repositories.UsersRepositoryInterface,
	notifierSrv notifier) *AuthHandler {
	return &AuthHandler{
		appConfig: appConfig,
		userRepo:  userRepo,
		notifier:  notifierSrv,
	}
}

// GetURL responds with the URL for OAuth 2.0 provider's consent page
func (h *AuthHandler) GetURL(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	respondJSON(w, struct {
		URL string
	}{
		URL: h.appConfig.GoogleOauth.AuthCodeURL(state, oauth2.AccessTypeOffline),
	}, http.StatusOK)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	oauthState := generateStateOauthCookie(w)
	u := h.appConfig.GoogleOauth.AuthCodeURL(oauthState)
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
}

func generateStateOauthCookie(w http.ResponseWriter) string {
	var expiration = time.Now().Add(365 * 24 * time.Hour)

	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{
		Name:    "oauthstate",
		Value:   state,
		Expires: expiration,
	}
	http.SetCookie(w, &cookie)

	return state
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	var codePayload AuthCodePayload
	err := json.NewDecoder(r.Body).Decode(&codePayload)
	if err != nil {
		respondInternalError(w)
		return
	}

	token, err := h.appConfig.GoogleOauth.Exchange(context.Background(), codePayload.Code)
	if err != nil {
		respondInternalError(w)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		respondInternalError(w)
		return
	}

	verifier := h.appConfig.OIDCProvider.Verifier(&oidc.Config{
		ClientID: h.appConfig.GoogleOauth.ClientID,
	})

	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		respondInternalError(w)
		return
	}

	var idTokenClaims struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := idToken.Claims(&idTokenClaims); err != nil {
		respondInternalError(w)
		return
	}

	_, _, err = h.userRepo.FindOrCreateUser(r.Context(), &repositories.User{
		ID:      idToken.Subject,
		Name:    idTokenClaims.Name,
		Email:   idTokenClaims.Email,
		Picture: idTokenClaims.Picture,
	})
	if err != nil {
		respondInternalError(w)
		return
	}

	respondJSON(w, struct {
		Token string
	}{
		Token: rawIDToken,
	}, http.StatusOK)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	token, err := h.appConfig.GoogleOauth.Exchange(context.Background(), code)
	if err != nil {
		respondInternalError(w)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		respondInternalError(w)
		return
	}

	verifier := h.appConfig.OIDCProvider.Verifier(&oidc.Config{
		ClientID: h.appConfig.GoogleOauth.ClientID,
	})

	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		respondInternalError(w)
		return
	}

	var idTokenClaims struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := idToken.Claims(&idTokenClaims); err != nil {
		respondInternalError(w)
		return
	}

	_, _, err = h.userRepo.FindOrCreateUser(r.Context(), &repositories.User{
		ID:      idToken.Subject,
		Name:    idTokenClaims.Name,
		Email:   idTokenClaims.Email,
		Picture: idTokenClaims.Picture,
	})
	if err != nil {
		respondInternalError(w)
		return
	}

	respondJSON(w, struct {
		Token string
	}{
		Token: rawIDToken,
	}, http.StatusOK)
}

func (h *AuthHandler) FindCreateUser(w http.ResponseWriter, r *http.Request) {
	platform := parsePlatformHeader(r.Header.Get("platform"))
	verifier := h.appConfig.OIDCProvider.Verifier(&oidc.Config{
		ClientID: h.appConfig.GetPlatformClientID(platform),
	})

	rawIDToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		respondInternalError(w)
		return
	}

	var idTokenClaims struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := idToken.Claims(&idTokenClaims); err != nil {
		respondInternalError(w)
		return
	}

	user, created, err := h.userRepo.FindOrCreateUser(r.Context(), &repositories.User{
		ID:      idToken.Subject,
		Name:    idTokenClaims.Name,
		Email:   idTokenClaims.Email,
		Picture: idTokenClaims.Picture,
	})

	if created == true && user != nil {
		go func() {
			userJSON, _ := json.Marshal(user)
			h.notifier.messageAll(usersTopic, map[string]string{
				"user": string(userJSON),
			})
		}()
	}

	respondJSON(w, user, http.StatusOK)
}
