package cluster

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/zxh326/kite/pkg/common"
	"github.com/zxh326/kite/pkg/model"
)

// newGetClustersContext builds a gin context with the given user and a
// ClusterManager holding two live clusters plus one errored one, so the
// per-user filtering of GetClusters can be asserted.
func newGetClustersContext(t *testing.T, user model.User) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	cm := &ClusterManager{
		clusters: map[string]*ClientSet{
			"sealos-alice-ws-a": {Name: "sealos-alice-ws-a"},
			"sealos-bob-ws-b":   {Name: "sealos-bob-ws-b"},
		},
		errors: map[string]string{
			"sealos-eve-ws-c": "failed to wait for cache sync",
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/clusters", nil)
	c.Set("user", user)

	cm.GetClusters(c)
	return c, w
}

func clusterNamesFromBody(t *testing.T, w *httptest.ResponseRecorder) []string {
	t.Helper()
	var infos []common.ClusterInfo
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &infos))
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		names = append(names, info.Name)
	}
	return names
}

// TestGetClustersFiltersPerUser locks in the requirement that a regular
// Sealos user only ever sees their own workspace cluster in the selector.
func TestGetClustersFiltersPerUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	regularUser := model.User{
		Username: "sealos-alice",
		Roles: []common.Role{
			{
				Name:       "sealos-role-alice",
				Clusters:   []string{"sealos-alice-ws-a"},
				Namespaces: []string{"ws-a"},
				Resources:  []string{"*"},
				Verbs:      []string{"*"},
			},
		},
	}

	_, w := newGetClustersContext(t, regularUser)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"sealos-alice-ws-a"}, clusterNamesFromBody(t, w))
}

// TestGetClustersAdminSeesEverything locks in the counterpart: an admin role
// with wildcard clusters sees every cluster, including errored ones.
func TestGetClustersAdminSeesEverything(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminUser := model.User{
		Username: "admin",
		Roles: []common.Role{
			{
				Name:       "admin",
				Clusters:   []string{"*"},
				Namespaces: []string{"*"},
				Resources:  []string{"*"},
				Verbs:      []string{"*"},
			},
		},
	}

	_, w := newGetClustersContext(t, adminUser)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t,
		[]string{"sealos-alice-ws-a", "sealos-bob-ws-b", "sealos-eve-ws-c"},
		clusterNamesFromBody(t, w),
	)
}
