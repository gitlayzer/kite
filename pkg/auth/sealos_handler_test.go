package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zxh326/kite/pkg/common"
	"github.com/zxh326/kite/pkg/model"
	"gorm.io/gorm"
)

func Test_buildSealosClusterUpdates(t *testing.T) {
	t.Run("unchanged cluster has no updates", func(t *testing.T) {
		cluster := &model.Cluster{
			Description:   "Managed by Sealos SSO (namespace: ns-a)",
			Config:        model.SecretString("same-kubeconfig"),
			PrometheusURL: "http://prometheus",
			InCluster:     false,
			Enable:        true,
		}

		updates := buildSealosClusterUpdates(cluster, "Managed by Sealos SSO (namespace: ns-a)", "same-kubeconfig", "http://prometheus")

		assert.Empty(t, updates)
	})

	t.Run("changed kubeconfig is updated", func(t *testing.T) {
		cluster := &model.Cluster{
			Description:   "Managed by Sealos SSO",
			Config:        model.SecretString("old-kubeconfig"),
			PrometheusURL: "http://prometheus",
			InCluster:     false,
			Enable:        true,
		}

		updates := buildSealosClusterUpdates(cluster, "Managed by Sealos SSO", "new-kubeconfig", "http://prometheus")

		assert.Equal(t, model.SecretString("new-kubeconfig"), updates["config"])
		assert.Len(t, updates, 1)
	})

	t.Run("missing defaults and disabled flags are repaired", func(t *testing.T) {
		cluster := &model.Cluster{
			Description:   "old description",
			Config:        model.SecretString("same-kubeconfig"),
			PrometheusURL: " ",
			InCluster:     true,
			Enable:        false,
		}

		updates := buildSealosClusterUpdates(cluster, "Managed by Sealos SSO", "same-kubeconfig", "http://prometheus")

		assert.Equal(t, "Managed by Sealos SSO", updates["description"])
		assert.Equal(t, false, updates["in_cluster"])
		assert.Equal(t, true, updates["enable"])
		assert.Equal(t, "http://prometheus", updates["prometheus_url"])
		assert.NotContains(t, updates, "config")
	})
}

func Test_buildSealosRoleNamespaces(t *testing.T) {
	originalExempt := common.NamespaceScopeExemptNamespaces
	t.Cleanup(func() {
		common.NamespaceScopeExemptNamespaces = originalExempt
	})

	t.Run("empty namespace grants all namespaces", func(t *testing.T) {
		common.NamespaceScopeExemptNamespaces = map[string]struct{}{}
		assert.Equal(t, []string{"*"}, buildSealosRoleNamespaces(""))
	})

	t.Run("regular namespace is namespace scoped", func(t *testing.T) {
		common.NamespaceScopeExemptNamespaces = map[string]struct{}{}
		assert.Equal(t, []string{"default"}, buildSealosRoleNamespaces("default"))
	})

	t.Run("exempt namespace grants all namespaces", func(t *testing.T) {
		common.NamespaceScopeExemptNamespaces = map[string]struct{}{
			"ns-admin": {},
		}
		assert.Equal(t, []string{"*"}, buildSealosRoleNamespaces("ns-admin"))
	})

	t.Run("exempt namespace matching is case insensitive and trim aware", func(t *testing.T) {
		common.NamespaceScopeExemptNamespaces = map[string]struct{}{
			"ns-admin": {},
		}
		assert.Equal(t, []string{"*"}, buildSealosRoleNamespaces(" NS-ADMIN "))
	})
}

func TestSyncSealosAdminRoleAssignment(t *testing.T) {
	useTestSealosAuthDB(t)

	originalExempt := common.NamespaceScopeExemptNamespaces
	t.Cleanup(func() {
		common.NamespaceScopeExemptNamespaces = originalExempt
	})

	require.NoError(t, model.InitDefaultRole())
	common.NamespaceScopeExemptNamespaces = map[string]struct{}{
		"ns-admin": {},
	}

	require.NoError(t, syncSealosAdminRoleAssignment("ns-admin", "sealos-admin"))
	require.NoError(t, syncSealosAdminRoleAssignment("ns-admin", "other-admin"))

	adminRole, err := model.GetRoleByName(model.DefaultAdminRole.Name)
	require.NoError(t, err)

	var adminAssignments int64
	require.NoError(t, model.DB.Model(&model.RoleAssignment{}).Where(
		"role_id = ? AND subject_type = ? AND subject = ?",
		adminRole.ID,
		model.SubjectTypeUser,
		"sealos-admin",
	).Count(&adminAssignments).Error)
	assert.EqualValues(t, 1, adminAssignments)

	require.NoError(t, syncSealosAdminRoleAssignment("default", "sealos-admin"))

	require.NoError(t, model.DB.Model(&model.RoleAssignment{}).Where(
		"role_id = ? AND subject_type = ? AND subject = ?",
		adminRole.ID,
		model.SubjectTypeUser,
		"sealos-admin",
	).Count(&adminAssignments).Error)
	assert.EqualValues(t, 0, adminAssignments)

	var otherAssignments int64
	require.NoError(t, model.DB.Model(&model.RoleAssignment{}).Where(
		"role_id = ? AND subject_type = ? AND subject = ?",
		adminRole.ID,
		model.SubjectTypeUser,
		"other-admin",
	).Count(&otherAssignments).Error)
	assert.EqualValues(t, 1, otherAssignments)

	require.NoError(t, syncSealosAdminRoleAssignment("default", "sealos-regular"))

	var regularAssignments int64
	require.NoError(t, model.DB.Model(&model.RoleAssignment{}).Where(
		"subject = ?",
		"sealos-regular",
	).Count(&regularAssignments).Error)
	assert.EqualValues(t, 0, regularAssignments)
}

func useTestSealosAuthDB(t *testing.T) {
	t.Helper()

	originalDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Role{}, &model.RoleAssignment{}))
	model.DB = db

	t.Cleanup(func() {
		model.DB = originalDB
	})
}

type fakeSealosClusterSyncer struct {
	hasCluster   bool
	triggeredFor []string
}

func (f *fakeSealosClusterSyncer) HasCluster(string) bool { return f.hasCluster }

func (f *fakeSealosClusterSyncer) TriggerSyncForCluster(name string) {
	f.triggeredFor = append(f.triggeredFor, name)
}

func TestSealosLoginTriggersAsyncClientBuild(t *testing.T) {
	setupStaleCleanupTestDB(t)

	originalEnabled := common.SealosAuthEnabled
	originalSecret := common.SealosJWTSecret
	originalExempt := common.NamespaceScopeExemptNamespaces
	t.Cleanup(func() {
		common.SealosAuthEnabled = originalEnabled
		common.SealosJWTSecret = originalSecret
		common.NamespaceScopeExemptNamespaces = originalExempt
	})
	common.SealosAuthEnabled = true
	common.SealosJWTSecret = "sealos-test-secret"
	common.NamespaceScopeExemptNamespaces = map[string]struct{}{}

	// Hydration normally runs in a background goroutine; run it
	// synchronously so assertions on the derived rows do not race the test
	// database teardown.
	originalHydrationHook := sealosHydrationHook
	sealosHydrationHook = func(f func()) { f() }
	t.Cleanup(func() { sealosHydrationHook = originalHydrationHook })

	// syncSealosAdminRoleAssignment looks up the built-in admin role even for
	// non-exempt workspaces (to strip a stale assignment), so seed it like
	// rbac.InitRBAC does in a real deployment.
	adminRole := model.DefaultAdminRole
	require.NoError(t, model.DB.Create(&adminRole).Error)

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, SealosTokenClaims{
		UserID:      "user-1",
		WorkspaceID: "ns-a",
	}).SignedString([]byte(common.SealosJWTSecret))
	require.NoError(t, err)

	// The fake has no live client and cannot build one: login must still
	// return immediately after kicking the build off. The previous handler
	// blocked here for up to 5s waiting for the client construction.
	syncer := &fakeSealosClusterSyncer{hasCluster: false}
	handler := &AuthHandler{
		manager:        NewOAuthManager(),
		clusterManager: syncer,
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := fmt.Sprintf(
		`{"token":%q,"kubeconfig":"dGVzdA==","user":{"userId":"user-1","nsid":"ns-a"}}`,
		signed,
	)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login/sealos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SealosLogin(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, []string{buildSealosClusterName("user-1", "ns-a")}, syncer.triggeredFor)
	assert.False(t, syncer.hasCluster)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, buildSealosClusterName("user-1", "ns-a"), resp["cluster"])

	// The fast path itself registers nothing, but the (here synchronous)
	// hydration must still derive the user and role rows in the background.
	userObj, ok := resp["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, buildSealosUsername("user-1"), userObj["username"])
	_, err = model.GetUserByUsername(buildSealosUsername("user-1"))
	require.NoError(t, err)
	_, err = model.GetRoleByName(buildSealosRoleName("user-1"))
	require.NoError(t, err)
}
