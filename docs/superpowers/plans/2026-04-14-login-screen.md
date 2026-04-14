# Login Screen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `spec/screens/login.md` — a public `GET /screens/login` endpoint that returns a standalone SDUI login form.

**Architecture:** New `internal/login/` package with a pure builder and a thin Gin handler that reads `Accept-Language`. The screen is static (no backend call), wired in the server's public route zone alongside `/health` and `/actions/login`. i18n keys added to `locales/en.json` and `locales/es.json`.

**Tech Stack:** Go, Gin, testify, `internal/components` helpers, `internal/i18n`.

---

## File Structure

**Create:**
- `internal/login/builder.go` — `BuildScreen(lang string) components.Component`.
- `internal/login/builder_test.go` — unit tests covering the acceptance criteria.
- `internal/login/handler.go` — Gin GET handler; reads `Accept-Language`; returns the tree.

**Modify:**
- `locales/en.json` — add eight `auth.*` keys.
- `locales/es.json` — same keys in Spanish.
- `internal/server/server.go` — register `GET /screens/login` in the public route section.

---

### Task 1: i18n keys for the login screen

**Files:**
- Modify: `locales/en.json`
- Modify: `locales/es.json`

- [ ] **Step 1: Update `locales/en.json`**

Replace its contents with:

```json
{
  "app": {
    "name": "VK Investment Tracker"
  },
  "nav": {
    "portfolio": "Portfolio",
    "assets": "Assets",
    "trades": "Trades",
    "snapshots": "Snapshots",
    "import": "Import",
    "analysis": "Analysis",
    "logout": "Log out"
  },
  "auth": {
    "login_title": "Log in",
    "email_label": "Email",
    "email_placeholder": "you@example.com",
    "password_label": "Password",
    "password_placeholder": "Your password",
    "submit": "Log in",
    "no_account_prompt": "Don't have an account?",
    "register_link": "Sign up"
  },
  "home": {
    "welcome_title": "Welcome to VK Investment Tracker",
    "subtitle": "This is a scaffolded middleend serving SDUI components."
  }
}
```

- [ ] **Step 2: Update `locales/es.json`**

Replace its contents with:

```json
{
  "app": {
    "name": "VK Investment Tracker"
  },
  "nav": {
    "portfolio": "Portafolio",
    "assets": "Activos",
    "trades": "Operaciones",
    "snapshots": "Snapshots",
    "import": "Importar",
    "analysis": "Análisis",
    "logout": "Cerrar sesión"
  },
  "auth": {
    "login_title": "Iniciar sesión",
    "email_label": "Correo",
    "email_placeholder": "vos@ejemplo.com",
    "password_label": "Contraseña",
    "password_placeholder": "Tu contraseña",
    "submit": "Ingresar",
    "no_account_prompt": "¿No tenés cuenta?",
    "register_link": "Registrarme"
  },
  "home": {
    "welcome_title": "Bienvenido a VK Investment Tracker",
    "subtitle": "Este es un middleend scaffoldeado que sirve componentes SDUI."
  }
}
```

- [ ] **Step 3: Verify existing tests still pass**

Run: `go test ./... -count=1`
Expected: all 50 tests still pass (shell tests load locales via `init()`).

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/es.json
git commit -m "feat(i18n): add auth.* keys for login screen"
```

---

### Task 2: Login screen builder

**Files:**
- Create: `internal/login/builder.go`
- Create: `internal/login/builder_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/login/builder_test.go`:

```go
package login

import (
	"path/filepath"
	"testing"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = i18n.Load(filepath.Join("..", "..", "locales"))
}

func TestBuildScreen_RootIsLoginScreen(t *testing.T) {
	s := BuildScreen("en")
	assert.Equal(t, "screen", s.Type)
	assert.Equal(t, "login", s.ID)
}

func TestBuildScreen_ContainsCardWithLogoTitleForm(t *testing.T) {
	s := BuildScreen("en")

	card := findDescendantByType(s, "card")
	require.NotNil(t, card, "card should be present in the tree")

	assert.NotNil(t, findDescendantByID(*card, "login-logo"), "logo image missing")

	title := findDescendantByID(*card, "login-title")
	require.NotNil(t, title, "title missing")
	assert.Equal(t, "text", title.Type)
	assert.Equal(t, "Log in", title.Props["content"])

	form := findDescendantByType(*card, "form")
	require.NotNil(t, form, "form missing")
	assert.Equal(t, "login-form", form.ID)
}

func TestBuildScreen_EmailAndPasswordInputsRequired(t *testing.T) {
	s := BuildScreen("en")

	email := findDescendantByID(s, "login-email")
	require.NotNil(t, email)
	assert.Equal(t, "input", email.Type)
	assert.Equal(t, "email", email.Props["input_type"])
	assert.Equal(t, "email", email.Props["name"])
	assert.Equal(t, true, email.Props["required"])

	password := findDescendantByID(s, "login-password")
	require.NotNil(t, password)
	assert.Equal(t, "input", password.Type)
	assert.Equal(t, "password", password.Props["input_type"])
	assert.Equal(t, "password", password.Props["name"])
	assert.Equal(t, true, password.Props["required"])
}

func TestBuildScreen_SubmitButtonHasSubmitAction(t *testing.T) {
	s := BuildScreen("en")

	btn := findDescendantByID(s, "login-submit")
	require.NotNil(t, btn)
	require.Len(t, btn.Actions, 1)
	a := btn.Actions[0]
	assert.Equal(t, "click", a.Trigger)
	assert.Equal(t, "submit", a.Type)
	assert.Equal(t, "/actions/login", a.Endpoint)
	assert.Equal(t, "POST", a.Method)
	assert.Equal(t, "login-form", a.TargetID)
}

func TestBuildScreen_RegisterLinkNavigates(t *testing.T) {
	s := BuildScreen("en")

	btn := findDescendantByID(s, "register-link")
	require.NotNil(t, btn)
	require.Len(t, btn.Actions, 1)
	a := btn.Actions[0]
	assert.Equal(t, "click", a.Trigger)
	assert.Equal(t, "navigate", a.Type)
	assert.Equal(t, "/screens/register", a.URL)
	assert.Equal(t, "self", a.Target)
}

func TestBuildScreen_NoShellSlots(t *testing.T) {
	s := BuildScreen("en")
	for _, slot := range []string{"nav_header", "nav_main", "nav_footer", "bottombar", "content_slot"} {
		assert.Nil(t, findDescendantByType(s, slot), "shell slot %q should not appear", slot)
	}
}

func TestBuildScreen_LabelsTranslated(t *testing.T) {
	en := BuildScreen("en")
	es := BuildScreen("es")

	enTitle := findDescendantByID(en, "login-title")
	esTitle := findDescendantByID(es, "login-title")
	require.NotNil(t, enTitle)
	require.NotNil(t, esTitle)
	assert.Equal(t, "Log in", enTitle.Props["content"])
	assert.Equal(t, "Iniciar sesión", esTitle.Props["content"])
}

func TestBuildScreen_UnknownLanguageFallsBackToEnglish(t *testing.T) {
	s := BuildScreen("zz")
	title := findDescendantByID(s, "login-title")
	require.NotNil(t, title)
	assert.Equal(t, "Log in", title.Props["content"])
}

func TestBuildScreen_RootColumnCentersContent(t *testing.T) {
	s := BuildScreen("en")
	root := findDescendantByID(s, "login-root")
	require.NotNil(t, root)
	assert.Equal(t, "column", root.Type)
	assert.Equal(t, "center", root.Props["align_items"])
	assert.Equal(t, "center", root.Props["justify_items"])
}

// helpers

func findDescendantByType(c components.Component, typ string) *components.Component {
	if c.Type == typ {
		return &c
	}
	for i := range c.Children {
		if found := findDescendantByType(c.Children[i], typ); found != nil {
			return found
		}
	}
	return nil
}

func findDescendantByID(c components.Component, id string) *components.Component {
	if c.ID == id {
		return &c
	}
	for i := range c.Children {
		if found := findDescendantByID(c.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/login/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement the builder**

Create `internal/login/builder.go`:

```go
package login

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildScreen builds the standalone login screen component tree.
// The screen has no shell — it renders on its own, full-viewport.
func BuildScreen(lang string) components.Component {
	emailInput := components.InputFull(
		"login-email", "email", "email",
		i18n.T(lang, "auth.email_label"),
		i18n.T(lang, "auth.email_placeholder"),
		"", true, false, 0,
	)

	passwordInput := components.InputFull(
		"login-password", "password", "password",
		i18n.T(lang, "auth.password_label"),
		i18n.T(lang, "auth.password_placeholder"),
		"", true, false, 0,
	)

	submit := components.Button(
		"login-submit", i18n.T(lang, "auth.submit"),
		components.Submit("/actions/login", "POST", "login-form"),
	)

	form := components.Form("login-form",
		components.ColumnWithGap("login-fields", "12px",
			emailInput,
			passwordInput,
			submit,
		),
	)

	registerRow := components.Row("register-row", []string{"auto", "auto"},
		components.Text("register-prompt", i18n.T(lang, "auth.no_account_prompt"), "sm", "normal"),
		components.ButtonFull(
			"register-link", i18n.T(lang, "auth.register_link"),
			"", "link", "solid",
			components.Navigate("/screens/register"),
		),
	)

	logo := components.Image("login-logo", "/logo.svg", i18n.T(lang, "app.name"))
	title := components.Text("login-title", i18n.T(lang, "auth.login_title"), "xl", "bold")

	card := components.Card("login-card",
		components.ColumnWithGap("login-content", "16px",
			logo,
			title,
			form,
			registerRow,
		),
	)

	root := components.Column("login-root", card)
	root.Props["align_items"] = "center"
	root.Props["justify_items"] = "center"

	return components.Screen("login", i18n.T(lang, "auth.login_title"), root)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/login/... -v`
Expected: PASS (9 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/login/builder.go internal/login/builder_test.go
git commit -m "feat(login): builder for standalone SDUI login screen"
```

---

### Task 3: Login screen HTTP handler

**Files:**
- Create: `internal/login/handler.go`
- Create: `internal/login/handler_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/login/handler_test.go`:

```go
package login

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/screens/login", NewHandler().Get)
	return r
}

func TestHandler_Returns200WithoutAuth(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest("GET", "/screens/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ReturnsLoginScreen(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest("GET", "/screens/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "screen", body["type"])
	assert.Equal(t, "login", body["id"])
}

func TestHandler_UsesAcceptLanguage(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest("GET", "/screens/login", nil)
	req.Header.Set("Accept-Language", "es")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Iniciar sesión")
}

func TestHandler_DefaultsToEnglishWhenNoAcceptLanguage(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest("GET", "/screens/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, w.Body.String(), "Log in")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/login/... -run TestHandler -v`
Expected: FAIL — `NewHandler` undefined.

- [ ] **Step 3: Implement the handler**

Create `internal/login/handler.go`:

```go
package login

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Handler serves GET /screens/login.
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// Get returns the login screen component tree. Public — no auth required.
func (h *Handler) Get(c *gin.Context) {
	lang := parseLang(c)
	c.JSON(http.StatusOK, BuildScreen(lang))
}

func parseLang(c *gin.Context) string {
	header := c.GetHeader("Accept-Language")
	if header == "" {
		return "en"
	}
	parts := strings.SplitN(header, ",", 2)
	lang := strings.SplitN(parts[0], "-", 2)[0]
	lang = strings.SplitN(lang, ";", 2)[0]
	return strings.TrimSpace(lang)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/login/... -v`
Expected: PASS (all 9 builder + 4 handler tests).

- [ ] **Step 5: Commit**

```bash
git add internal/login/handler.go internal/login/handler_test.go
git commit -m "feat(login): GET /screens/login handler"
```

---

### Task 4: Wire the login screen as a public route

**Files:**
- Modify: `internal/server/server.go`
- Modify: `internal/server/server_test.go`

- [ ] **Step 1: Add a failing test**

Append to `internal/server/server_test.go`:

```go
func TestServer_LoginScreenIsPublic(t *testing.T) {
	s := New(testConfig())
	req := httptest.NewRequest("GET", "/screens/login", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/server/... -run TestServer_LoginScreenIsPublic -v`
Expected: FAIL — route returns 404 or 401.

- [ ] **Step 3: Wire the route in the server**

In `internal/server/server.go`, locate the public routes section (just after `s.router.GET("/health", s.healthHandler)` and the login/register action routes). Replace the existing `setupRoutes` function with:

```go
func (s *Server) setupRoutes() {
	// Public routes (no auth).
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/screens/login", login.NewHandler().Get)

	authClient := auth.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
	s.router.POST("/actions/login", auth.NewLoginHandler(authClient).Post)
	s.router.POST("/actions/register", auth.NewRegisterHandler(authClient).Post)

	// Protected routes.
	leeway := time.Duration(s.cfg.JWTLeewaySeconds) * time.Second
	protected := s.router.Group("")
	protected.Use(auth.RequireAuth(s.cfg.JWTSecret, leeway, "/screens/login"))

	shellUC := shell.NewGetUseCase()
	shellHandler := shell.NewHandler(shellUC)
	protected.GET("/shell", shellHandler.Get)

	homeClient := home.NewClient(s.cfg.BackendURL)
	homeUC := home.NewGetUseCase(homeClient)
	homeHandler := home.NewHandler(homeUC)
	protected.GET("/screens/home", homeHandler.Get)
}
```

Also add `"github.com/project/vk-investment-middleend/internal/login"` to the imports — alphabetically between `internal/home` and `internal/shell`.

- [ ] **Step 4: Run tests**

Run: `go test ./... -count=1`
Expected: all tests PASS.

- [ ] **Step 5: Smoke-test the running server**

Run:

```bash
lsof -ti:8081 | xargs kill -9 2>/dev/null; sleep 1
./cli run >/tmp/srv.log 2>&1 &
sleep 2
echo "--- GET /screens/login (no token) ---"
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8081/screens/login
echo "--- GET /screens/login (es) body snippet ---"
curl -s http://localhost:8081/screens/login -H "Accept-Language: es" | head -c 200
echo
echo "--- GET /shell (no token — must still 401 with redirect) ---"
curl -s -w "\n%{http_code}\n" http://localhost:8081/shell
lsof -ti:8081 | xargs kill -9 2>/dev/null; true
```

Expected:
```
--- GET /screens/login (no token) ---
200
--- GET /screens/login (es) body snippet ---
{"type":"screen","id":"login", ... "Iniciar sesión" ...}
--- GET /shell (no token — must still 401 with redirect) ---
{"error":"unauthorized","redirect":"/screens/login"}
401
```

- [ ] **Step 6: Verify build and lint**

Run: `./cli build 2>&1 | tail -1 && ./cli lint 2>&1 | tail -1`
Expected: both success.

- [ ] **Step 7: Commit**

```bash
git add internal/server/server.go internal/server/server_test.go
git commit -m "feat(server): register public GET /screens/login"
```

---

## Self-Review Results

**Spec coverage check:**

| Spec acceptance criterion | Task |
|---|---|
| `GET /screens/login` returns 200 without `Authorization` | Task 3 `TestHandler_Returns200WithoutAuth` + Task 4 `TestServer_LoginScreenIsPublic` + smoke test |
| Response is `type: screen`, `id: login` | Task 2 `TestBuildScreen_RootIsLoginScreen`, Task 3 `TestHandler_ReturnsLoginScreen` |
| Tree contains card, logo, title, form, email+password (required), submit | Task 2 `TestBuildScreen_ContainsCardWithLogoTitleForm`, `TestBuildScreen_EmailAndPasswordInputsRequired` |
| Submit action `submit /actions/login POST login-form` | Task 2 `TestBuildScreen_SubmitButtonHasSubmitAction` |
| Register button `navigate /screens/register` | Task 2 `TestBuildScreen_RegisterLinkNavigates` |
| All strings via i18n | Task 1 (keys exist), Task 2 builder uses `i18n.T` for every label/placeholder/title/prompt/link/logo-alt |
| `Accept-Language: es` returns Spanish | Task 2 `TestBuildScreen_LabelsTranslated`, Task 3 `TestHandler_UsesAcceptLanguage` |
| Unknown language → English fallback | Task 2 `TestBuildScreen_UnknownLanguageFallsBackToEnglish` |
| No shell slots in tree | Task 2 `TestBuildScreen_NoShellSlots` |
| Centering via `align_items`/`justify_items` on root column | Task 2 `TestBuildScreen_RootColumnCentersContent` |

All acceptance criteria covered.

**Placeholder scan:** none.

**Type consistency:**
- `BuildScreen(lang string) components.Component` — same signature in Task 2 builder, Task 2 tests, Task 3 handler.
- `NewHandler()` no-arg constructor — Task 3 impl and test use it identically; Task 4 wiring uses `login.NewHandler().Get` consistently.
- Component IDs (`login-root`, `login-card`, `login-logo`, `login-title`, `login-form`, `login-fields`, `login-email`, `login-password`, `login-submit`, `register-row`, `register-prompt`, `register-link`) are consistent between builder and tests.
- i18n keys in Task 1 JSON match the `i18n.T` calls in Task 2 builder (`auth.login_title`, `auth.email_label`, `auth.email_placeholder`, `auth.password_label`, `auth.password_placeholder`, `auth.submit`, `auth.no_account_prompt`, `auth.register_link`, `app.name`).
