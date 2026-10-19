package httpapi

// Login com Google usando OAuth2 "Authorization Code com PKCE".
//
// O fluxo tem duas rotas e o navegador do usuário passa pelas duas:
//
//  1. GET /auth/google — sorteamos um `state` (anti-CSRF) e um
//     `code_verifier` (PKCE), guardamos os dois num cookie HttpOnly e
//     redirecionamos para a tela de consentimento do Google.
//  2. GET /auth/google/callback — o Google volta com um `code`. Conferimos
//     o `state` contra o cookie, trocamos o código pelo token (provando
//     que somos quem iniciou, via code_verifier), lemos o perfil e
//     emitimos o MESMO JWT do login por senha.
//
// Sem as credenciais configuradas as duas rotas respondem 503, então o
// projeto sobe normalmente sem elas.

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/arthursa-source/forms/internal/auth"
	"github.com/arthursa-source/forms/internal/store"
	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Nome do cookie temporário que carrega o state e o code_verifier entre
// as duas rotas, e por quanto tempo ele vale (o usuário tem esse tempo
// para concluir o consentimento).
const (
	oauthCookieName = "forms_google_oauth"
	oauthCookieTTL  = 10 * time.Minute
)

// Endpoint OIDC do Google com os dados do perfil. Pedimos o escopo
// "openid email profile" e aqui lemos o resultado.
const googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"

// ---- formatos ---------------------------------------------------------

// Respostas destas rotas são redirecionamentos: só header Location,
// status 302 e o cookie. O campo `Status` é lido pelo huma.
type redirectOutput struct {
	Status    int
	Location  string      `header:"Location"`
	SetCookie http.Cookie `header:"Set-Cookie"`
}

type googleCallbackInput struct {
	Code  string `query:"code" doc:"código de autorização devolvido pelo Google"`
	State string `query:"state" doc:"deve bater com o state guardado no cookie"`
	// O Google manda ?error=access_denied quando o usuário cancela.
	Error  string `query:"error" doc:"preenchido quando o usuário recusa o consentimento"`
	Cookie string `cookie:"forms_google_oauth"`
}

// ---- registro ---------------------------------------------------------

func (s *Server) registerGoogleAuth(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "googleLogin",
		Method:        http.MethodGet,
		Path:          "/auth/google",
		Summary:       "Inicia o login com Google",
		Description:   "Redireciona (302) para a tela de consentimento do Google. Responde 503 quando as credenciais do Google não estão configuradas.",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusFound,
	}, s.handleGoogleLogin)

	huma.Register(api, huma.Operation{
		OperationID:   "googleCallback",
		Method:        http.MethodGet,
		Path:          "/auth/google/callback",
		Summary:       "Recebe o retorno do Google e emite o JWT",
		Description:   "Redireciona (302) para o frontend com o token no fragmento da URL (#token=...). Em caso de erro, redireciona com #error=...",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusFound,
	}, s.handleGoogleCallback)
}

// googleEnabled: sem as três credenciais não há fluxo possível.
func (s *Server) googleEnabled() bool {
	return s.GoogleClientID != "" && s.GoogleClientSecret != "" && s.GoogleRedirectURL != ""
}

// oauthConfig monta a configuração do pacote oauth2 com o endpoint do
// Google. Os escopos são os mínimos: identificar a conta e ler o e-mail.
func (s *Server) oauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.GoogleClientID,
		ClientSecret: s.GoogleClientSecret,
		RedirectURL:  s.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

// ---- handlers ---------------------------------------------------------

func (s *Server) handleGoogleLogin(_ context.Context, _ *struct{}) (*redirectOutput, error) {
	if !s.googleEnabled() {
		return nil, huma.Error503ServiceUnavailable("login com Google não está configurado neste servidor")
	}

	// state protege contra CSRF: quem chamar a callback tem que apresentar
	// o mesmo valor que guardamos no cookie.
	state, err := randomToken()
	if err != nil {
		return nil, err
	}
	// code_verifier é o segredo do PKCE: mandamos para o Google apenas o
	// hash (S256ChallengeOption) e revelamos o original só na troca do
	// código. Assim um código interceptado não serve para mais ninguém.
	verifier := oauth2.GenerateVerifier()

	return &redirectOutput{
		Status: http.StatusFound,
		Location: s.oauthConfig().AuthCodeURL(state,
			oauth2.AccessTypeOnline,
			oauth2.S256ChallengeOption(verifier),
		),
		SetCookie: s.oauthCookie(state + "." + verifier),
	}, nil
}

func (s *Server) handleGoogleCallback(ctx context.Context, in *googleCallbackInput) (*redirectOutput, error) {
	if !s.googleEnabled() {
		return nil, huma.Error503ServiceUnavailable("login com Google não está configurado neste servidor")
	}

	token, expiresAt, err := s.googleSession(ctx, in)
	if err != nil {
		// Erros aqui são do navegador do usuário, não de uma chamada de
		// API: devolver JSON deixaria uma página branca. Voltamos para o
		// frontend com a mensagem no fragmento.
		return s.frontendRedirect("error=" + url.QueryEscape(err.Error())), nil
	}

	return s.frontendRedirect(fmt.Sprintf("token=%s&expiresAt=%s",
		url.QueryEscape(token),
		url.QueryEscape(expiresAt.Format(time.RFC3339)),
	)), nil
}

// googleSession faz todo o trabalho da callback e devolve o nosso JWT.
// Separado do handler para o handler só cuidar do redirecionamento.
func (s *Server) googleSession(ctx context.Context, in *googleCallbackInput) (string, time.Time, error) {
	if in.Error != "" {
		return "", time.Time{}, fmt.Errorf("o Google recusou o login: %s", in.Error)
	}

	state, verifier, ok := strings.Cut(in.Cookie, ".")
	if !ok || state == "" || verifier == "" {
		return "", time.Time{}, errors.New("sessão de login expirada, tente novamente")
	}
	// ConstantTimeCompare em vez de == para não vazar informação pelo
	// tempo da comparação.
	if subtle.ConstantTimeCompare([]byte(state), []byte(in.State)) != 1 {
		return "", time.Time{}, errors.New("state inválido, tente novamente")
	}
	if in.Code == "" {
		return "", time.Time{}, errors.New("o Google não devolveu o código de autorização")
	}

	cfg := s.oauthConfig()
	// Troca do código pelo access token, agora revelando o code_verifier.
	googleToken, err := cfg.Exchange(ctx, in.Code, oauth2.VerifierOption(verifier))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("não foi possível validar o login com o Google")
	}

	profile, err := fetchGoogleProfile(ctx, cfg, googleToken)
	if err != nil {
		return "", time.Time{}, err
	}

	user, err := s.userForGoogleProfile(ctx, profile)
	if err != nil {
		return "", time.Time{}, err
	}

	// Daqui para frente é igual ao login por senha: o mesmo JWT, lido
	// pelo mesmo middleware.
	return auth.IssueToken(s.JWTSecret, user.ID, user.Email)
}

// googleProfile é o recorte do userinfo que usamos.
type googleProfile struct {
	// "sub" é o id estável da conta no Google; o e-mail pode mudar.
	Sub   string `json:"sub"`
	Email string `json:"email"`
	// Ponteiro para diferenciar "false" de "campo ausente".
	EmailVerified *bool `json:"email_verified"`
}

func fetchGoogleProfile(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (googleProfile, error) {
	// cfg.Client devolve um http.Client que já anexa o access token.
	res, err := cfg.Client(ctx, token).Get(googleUserInfoURL)
	if err != nil {
		return googleProfile{}, fmt.Errorf("não foi possível ler o perfil no Google")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return googleProfile{}, fmt.Errorf("o Google respondeu %d ao pedir o perfil", res.StatusCode)
	}

	var p googleProfile
	if err := json.NewDecoder(res.Body).Decode(&p); err != nil {
		return googleProfile{}, fmt.Errorf("perfil do Google em formato inesperado")
	}
	if p.Sub == "" || p.Email == "" {
		return googleProfile{}, errors.New("a conta do Google não informou um e-mail")
	}
	// Vincular contas pelo e-mail só é seguro se o Google garantir que o
	// e-mail é daquela pessoa.
	if p.EmailVerified == nil || !*p.EmailVerified {
		return googleProfile{}, errors.New("o e-mail desta conta do Google não está verificado")
	}
	return p, nil
}

// userForGoogleProfile resolve as três situações possíveis: já entrou
// pelo Google antes, já tem conta de e-mail/senha com o mesmo e-mail, ou
// é a primeira vez.
func (s *Server) userForGoogleProfile(ctx context.Context, p googleProfile) (store.User, error) {
	user, err := s.Store.UserByGoogleID(ctx, p.Sub)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return store.User{}, err
	}

	// Mesmo e-mail já cadastrado: vincula as contas em vez de recusar.
	existing, err := s.Store.UserByEmail(ctx, p.Email)
	if err == nil {
		return s.Store.LinkGoogleID(ctx, existing.ID, p.Sub)
	}
	if !errors.Is(err, store.ErrNotFound) {
		return store.User{}, err
	}

	return s.Store.CreateGoogleUser(ctx, p.Email, p.Sub)
}

// ---- auxiliares -------------------------------------------------------

// frontendRedirect manda o navegador para a página de callback do
// frontend. Os dados vão no FRAGMENTO (#) de propósito: fragmento não é
// enviado ao servidor, então o token não aparece em log de acesso nem em
// header Referer.
func (s *Server) frontendRedirect(fragment string) *redirectOutput {
	return &redirectOutput{
		Status:   http.StatusFound,
		Location: strings.TrimSuffix(s.FrontendURL, "/") + "/auth/callback#" + fragment,
		// Cookie de valor vazio e MaxAge negativo: apaga o cookie do
		// fluxo, que já cumpriu sua função.
		SetCookie: s.expiredOAuthCookie(),
	}
}

func (s *Server) oauthCookie(value string) http.Cookie {
	return http.Cookie{
		Name:  oauthCookieName,
		Value: value,
		Path:  "/",
		// HttpOnly: JavaScript não lê, então um XSS não rouba o state.
		HttpOnly: true,
		// Lax permite que o cookie viaje no retorno do Google (navegação
		// de topo vinda de outro site), mas não em requisições de
		// terceiros embutidas.
		SameSite: http.SameSiteLaxMode,
		// Só exige HTTPS quando o próprio redirect é HTTPS; em
		// desenvolvimento o fluxo roda em http://localhost.
		Secure: strings.HasPrefix(s.GoogleRedirectURL, "https://"),
		MaxAge: int(oauthCookieTTL.Seconds()),
	}
}

func (s *Server) expiredOAuthCookie() http.Cookie {
	c := s.oauthCookie("")
	c.MaxAge = -1
	return c
}

// randomToken gera 32 bytes aleatórios em base64 sem padding — usado no
// state. crypto/rand é o gerador criptográfico.
func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("gerando valor aleatório: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
