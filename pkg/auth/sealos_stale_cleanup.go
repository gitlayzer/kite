package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/common"
	"github.com/zxh326/kite/pkg/model"
	"github.com/zxh326/kite/pkg/rbac"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

const (
	// staleCleanupInterval is how often the sweep runs. An hourly pass keeps
	// the residual list short without adding measurable database load.
	staleCleanupInterval = time.Hour
	// staleSweepBatchSize caps how many stale users one pass deletes, so a
	// huge backlog is drained gradually instead of in one blocking burst.
	staleSweepBatchSize = 50
)

// StartStaleSealosCleanup periodically removes Sealos auto-provisioned
// accounts (and the clusters/roles their login created) that have been
// inactive longer than KITE_SEALOS_STALE_TTL_DAYS. On multi-tenant Sealos
// platforms the cluster selector otherwise fills up with residual workspace
// entries whose kubeconfigs have long expired — which shows up as "Sync
// Error" noise for admins and as repeated failed client rebuilds in the
// cluster sync loop.
func StartStaleSealosCleanup(ctx context.Context, cm *cluster.ClusterManager) {
	ttlDays := common.SealosStaleCleanupTTLDays
	if ttlDays <= 0 {
		klog.Infof("sealos stale cleanup disabled (KITE_SEALOS_STALE_TTL_DAYS=%d)", ttlDays)
		return
	}
	klog.Infof("sealos stale cleanup enabled: ttl=%dd interval=%s", ttlDays, staleCleanupInterval)

	ticker := time.NewTicker(staleCleanupInterval)
	defer ticker.Stop()
	for {
		removed, err := SweepStaleSealosUsers(time.Now(), ttlDays, staleSweepBatchSize)
		if err != nil {
			klog.Warningf("sealos stale cleanup sweep failed: %v", err)
		}
		if removed > 0 {
			// Torn-down clusters must drop their informer clients and RBAC
			// must forget the deleted roles/assignments.
			cm.TriggerSync()
			if err := rbac.ForceSyncRolesConfig(); err != nil {
				klog.Warningf("sealos stale cleanup: failed to reload RBAC config: %v", err)
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// SweepStaleSealosUsers deletes up to limit Sealos auto-provisioned users
// whose last login is older than ttlDays (or who never logged in and were
// created longer ago than that), together with everything their login
// created: the auto-generated role, all their role assignments, and every
// sealos-* cluster they ever owned. It returns the number of deleted users.
func SweepStaleSealosUsers(now time.Time, ttlDays, limit int) (int, error) {
	cutoff := now.AddDate(0, 0, -ttlDays)

	var staleUsers []model.User
	err := model.DB.
		Where("provider = ?", sealosProvider).
		Where(
			model.DB.Where("last_login_at < ?", cutoff).
				Or("last_login_at IS NULL AND created_at < ?", cutoff),
		).
		Limit(limit).
		Find(&staleUsers).Error
	if err != nil {
		return 0, err
	}

	removed := 0
	for i := range staleUsers {
		u := &staleUsers[i]
		// Sealos users carry the platform user ID in Sub as "sealos:<id>";
		// anything else means the row was not auto-provisioned — skip it.
		userID, ok := strings.CutPrefix(u.Sub, sealosProvider+":")
		if !ok || strings.TrimSpace(userID) == "" {
			continue
		}

		clusterNames := map[string]struct{}{}

		// (a) Name-prefix match: cluster names are built as
		// "sealos-<userPart>-<workspacePart>", and because ensureSealosRole
		// keeps only the most recent cluster in the role, older workspaces
		// are orphaned from it and only findable by name.
		prefix := "sealos-" + sanitizeNamePart(userID) + "-"
		var namedClusters []model.Cluster
		if err := model.DB.Where("name LIKE ?", prefix+"%").Find(&namedClusters).Error; err != nil {
			return removed, err
		}
		for _, cl := range namedClusters {
			clusterNames[cl.Name] = struct{}{}
		}

		// (b) The auto-generated role also references the current cluster.
		roleName := buildSealosRoleName(userID)
		role, roleErr := model.GetRoleByName(roleName)
		if roleErr == nil {
			for _, name := range role.Clusters {
				if strings.HasPrefix(name, "sealos-") {
					clusterNames[name] = struct{}{}
				}
			}
		} else if !errors.Is(roleErr, gorm.ErrRecordNotFound) {
			return removed, roleErr
		}

		for name := range clusterNames {
			cl, err := model.GetClusterByName(name)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return removed, err
			}
			// Only auto-created sealos-* clusters may be swept; anything else
			// (e.g. manually imported) is left alone.
			if !strings.HasPrefix(cl.Name, "sealos-") {
				continue
			}
			if err := model.DeleteCluster(cl); err != nil {
				return removed, err
			}
			klog.Infof("sealos stale cleanup: deleted cluster %s (owner %s inactive)", cl.Name, u.Username)
		}

		// Drop every role assignment of this user, covering the auto role and
		// any built-in admin assignment from exempt workspaces.
		if err := model.DB.
			Where("subject_type = ? AND subject = ?", model.SubjectTypeUser, u.Username).
			Delete(&model.RoleAssignment{}).Error; err != nil {
			return removed, err
		}

		if roleErr == nil {
			if err := model.DB.Delete(role).Error; err != nil {
				return removed, err
			}
		}

		// DeleteUserByID also removes the user's resource history.
		if err := model.DeleteUserByID(u.ID); err != nil {
			return removed, err
		}
		removed++
		klog.Infof("sealos stale cleanup: deleted user %s with %d cluster(s)", u.Username, len(clusterNames))
	}

	return removed, nil
}
