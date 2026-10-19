package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zxh326/kite/pkg/common"
	"github.com/zxh326/kite/pkg/model"
	"github.com/zxh326/kite/pkg/rbac"
	"gorm.io/gorm"
)

func signSealosTestToken(t *testing.T, claims SealosTokenClaims) string {
	t.Helper()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(common.SealosJWTSecret))
	require.NoError(t, err)
	return signed
}

func sealosLoginJSONBody(t *testing.T, token, kubeconfig, userID, nsid string) string {
	t.Helper()
	return fmt.Sprintf(
		`{"token":%q,"kubeconfig":%q,"user":{"userId":%q,"nsid":%q}}`,
		token, kubeconfig, userID, nsid,
	)
}

func postSealosLogin(t *testing.T, h *AuthHandler, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login/sealos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.SealosLogin(c)
	var resp map[string]any
	if w.Body.Len() > 0 {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	}
	return w, resp
}

func runRequireAuth(t *testing.T, h *AuthHandler, jwtToken, path string) (*httptest.ResponseRecorder, model.User, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)
	c.Request.Header.Set("Authorization", "Bearer "+jwtToken)
	h.RequireAuth()(c)
	userValue, ok := c.Get("user")
	if !ok {
		return w, model.User{}, false
	}
	return w, userValue.(model.User), true
}

// setupSealosClaimsTest prepares globals, the test database and the built-in
// admin role for the Sealos claims tests.
func setupSealosClaimsTest(t *testing.T, exempt []string) {
	t.Helper()
	setupStaleCleanupTestDB(t)

	originalEnabled := common.SealosAuthEnabled
	originalSecret := common.SealosJWTSecret
	originalExempt := common.NamespaceScopeExemptNamespaces
	originalHook := sealosHydrationHook
	t.Cleanup(func() {
		common.SealosAuthEnabled = originalEnabled
		common.SealosJWTSecret = originalSecret
		common.NamespaceScopeExemptNamespaces = originalExempt
		sealosHydrationHook = originalHook
	})
	common.SealosAuthEnabled = true
	common.SealosJWTSecret = "sealos-test-secret"
	common.NamespaceScopeExemptNamespaces = map[string]struct{}{}
	for _, ns := range exempt {
		common.NamespaceScopeExemptNamespaces[ns] = struct{}{}
	}

	// syncSealosAdminRoleAssignment looks up the built-in admin role even
	// for non-exempt workspaces (to strip a stale assignment), so seed it
	// like rbac.InitRBAC does in a real deployment.
	adminRole := model.DefaultAdminRole
	require.NoError(t, model.DB.Create(&adminRole).Error)
}

func TestSealosLoginDefersUserRegistrationToBackground(t *testing.T) {
	setupSealosClaimsTest(t, nil)

	// Capture the hydration instead of running it: the login response must
	// be fully usable before any user or role row exists.
	var captured func()
	sealosHydrationHook = func(f func()) { captured = f }

	token := signSealosTestToken(t, SealosTokenClaims{UserID: "user-1", WorkspaceID: "ns-a"})
	handler := &AuthHandler{manager: NewOAuthManager()}
	w, resp := postSealosLogin(t, handler, sealosLoginJSONBody(t, token, "dGVzdA==", "user-1", "ns-a"))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, captured, "login must schedule the identity hydration")
	assert.Equal(t, "Bearer", resp["token_type"])
	assert.Equal(t, buildSealosClusterName("user-1", "ns-a"), resp["cluster"])
	userObj, ok := resp["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, buildSealosUsername("user-1"), userObj["username"])

	// The request path itself registered nothing.
	_, err := model.GetUserByUsername(buildSealosUsername("user-1"))
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Running the scheduled hydration derives the same artifacts the old
	// synchronous login wrote on the request path.
	captured()
	_, err = model.GetUserByUsername(buildSealosUsername("user-1"))
	require.NoError(t, err)
	_, err = model.GetRoleByName(buildSealosRoleName("user-1"))
	require.NoError(t, err)
}

func TestSealosClaimsSessionAuthorizesWithoutUserRow(t *testing.T) {
	setupSealosClaimsTest(t, nil)

	// Never run the hydration in this test: there must be no user row at
	// all, and the session must still authenticate and authorize.
	sealosHydrationHook = func(f func()) {}

	token := signSealosTestToken(t, SealosTokenClaims{UserID: "user-1", WorkspaceID: "ns-a"})
	handler := &AuthHandler{manager: NewOAuthManager()}
	w, resp := postSealosLogin(t, handler, sealosLoginJSONBody(t, token, "dGVzdA==", "user-1", "ns-a"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	accessToken, ok := resp["access_token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, accessToken)

	_, err := model.GetUserByUsername(buildSealosUsername("user-1"))
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	clusterName := buildSealosClusterName("user-1", "ns-a")
	_, user, authed := runRequireAuth(t, handler, accessToken, "/api/v1/pods")
	require.True(t, authed, "claims session must authenticate without a user row")
	assert.Equal(t, buildSealosUsername("user-1"), user.Username)
	assert.True(t, rbac.CanAccess(user, "pods", "get", clusterName, "ns-a"))
	assert.False(t, rbac.CanAccess(user, "pods", "get", clusterName, "ns-b"))
	assert.False(t, rbac.CanAccess(user, "nodes", "get", clusterName, "_all"))
}

func TestSealosLoginExemptWorkspaceGetsAdminClaims(t *testing.T) {
	setupSealosClaimsTest(t, []string{"ns-admin"})
	sealosHydrationHook = func(f func()) { f() }

	token := signSealosTestToken(t, SealosTokenClaims{UserID: "user-9", WorkspaceID: "ns-admin"})
	handler := &AuthHandler{manager: NewOAuthManager()}
	w, resp := postSealosLogin(t, handler, sealosLoginJSONBody(t, token, "dGVzdA==", "user-9", "ns-admin"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	accessToken, ok := resp["access_token"].(string)
	require.True(t, ok)

	clusterName := buildSealosClusterName("user-9", "ns-admin")
	_, user, authed := runRequireAuth(t, handler, accessToken, "/api/v1/nodes")
	require.True(t, authed)
	assert.True(t, rbac.UserHasRole(user, model.DefaultAdminRole.Name))
	assert.True(t, rbac.CanAccess(user, "nodes", "get", clusterName, "_all"))

	// The DB shadow kept the built-in admin assignment in sync.
	var count int64
	require.NoError(t, model.DB.Model(&model.RoleAssignment{}).
		Joins("JOIN roles ON roles.id = role_assignments.role_id").
		Where("roles.name = ? AND role_assignments.subject = ?", model.DefaultAdminRole.Name, buildSealosUsername("user-9")).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestSealosLoginRejectsDisabledUser(t *testing.T) {
	setupSealosClaimsTest(t, nil)
	username := buildSealosUsername("user-1")
	require.NoError(t, model.DB.Create(&model.User{
		Username: username,
		Provider: sealosProvider,
		Sub:      sealosProvider + ":user-1",
	}).Error)
	// Write the disabled flag explicitly: the column default (true) would
	// otherwise win over the zero value on insert.
	require.NoError(t, model.DB.Model(&model.User{}).
		Where("username = ?", username).
		Update("enabled", false).Error)
	t.Cleanup(func() { model.SetUsernameDisabled(username, false) })

	sealosHydrationHook = func(f func()) { t.Errorf("hydration must not run for a disabled user") }
	token := signSealosTestToken(t, SealosTokenClaims{UserID: "user-1", WorkspaceID: "ns-a"})
	handler := &AuthHandler{manager: NewOAuthManager()}
	w, _ := postSealosLogin(t, handler, sealosLoginJSONBody(t, token, "dGVzdA==", "user-1", "ns-a"))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireAuthRejectsDisabledSealosSession(t *testing.T) {
	setupSealosClaimsTest(t, nil)
	originalJWT := common.JwtSecret
	t.Cleanup(func() {
		common.JwtSecret = originalJWT
		model.SetUsernameDisabled("sealos-u9", false)
	})
	common.JwtSecret = "kite-test-secret"
	handler := &AuthHandler{manager: NewOAuthManager()}

	token, err := handler.manager.GenerateSealosJWT(
		&model.User{Username: "sealos-u9", Provider: sealosProvider, Sub: sealosProvider + ":user-9"},
		"sealos-c9", "ns-a", "ns-a", false,
	)
	require.NoError(t, err)

	_, _, authed := runRequireAuth(t, handler, token, "/api/v1/pods")
	require.True(t, authed)

	model.SetUsernameDisabled("sealos-u9", true)
	w, _, authed := runRequireAuth(t, handler, token, "/api/v1/pods")
	assert.False(t, authed)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSealosVirtualRolesEnforceWorkspaceScope(t *testing.T) {
	scoped := model.User{Username: "u", Roles: sealosVirtualRoles("c1", []string{"ns-a"}, false)}
	assert.True(t, rbac.CanAccess(scoped, "pods", "get", "c1", "ns-a"))
	assert.False(t, rbac.CanAccess(scoped, "pods", "get", "c1", "ns-b"))
	assert.False(t, rbac.CanAccess(scoped, "pods", "get", "c1", "_all"))
	assert.False(t, rbac.CanAccess(scoped, "pods", "get", "c2", "ns-a"))
	assert.False(t, rbac.UserHasRole(scoped, model.DefaultAdminRole.Name))

	global := model.User{Username: "a", Roles: sealosVirtualRoles("c1", []string{"*"}, true)}
	assert.True(t, rbac.CanAccess(global, "pods", "get", "c1", "ns-b"))
	assert.True(t, rbac.CanAccess(global, "nodes", "get", "c1", "_all"))
	assert.True(t, rbac.UserHasRole(global, model.DefaultAdminRole.Name))
}
