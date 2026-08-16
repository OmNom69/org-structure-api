package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/OmNom69/org-structure-api/internal/dto"
	"golang.org/x/sync/singleflight"
)

const (
	departmentTreeEpochKey         = "department-tree:epoch"
	maxDepartmentTreeCacheJSONSize = 1 << 20
	departmentTreeTTLJitterPercent = 10
)

type departmentTreeCache struct {
	store  CacheStore
	ttl    time.Duration
	jitter func(time.Duration) time.Duration
	group  singleflight.Group
}

func newDepartmentTreeCache(store CacheStore, ttl time.Duration) departmentTreeCache {
	return departmentTreeCache{
		store:  store,
		ttl:    ttl,
		jitter: newTTLJitter(time.Now().UnixNano()),
	}
}

func (s *DepartmentService) getDepartmentTreeCached(ctx context.Context, id uint, depth int, includeEmployees bool) (*dto.DepartmentTreeResponse, error) {
	epoch, err := readDepartmentTreeEpoch(ctx, s.treeCache.store)
	if err != nil {
		return s.loadDepartmentTree(ctx, id, depth, includeEmployees)
	}

	key := departmentTreeCacheKey(epoch, id, depth, includeEmployees)
	cached, found, err := s.readCachedDepartmentTree(
		ctx,
		key,
		id,
		depth,
		includeEmployees,
	)
	if err != nil {
		return s.loadDepartmentTree(ctx, id, depth, includeEmployees)
	}
	if found {
		return cached, nil
	}

	value, err, _ := s.treeCache.group.Do(key, func() (any, error) {
		return s.loadOrCacheDepartmentTree(ctx, key, id, depth, includeEmployees)
	})
	if err != nil {
		return nil, err
	}

	return value.(*dto.DepartmentTreeResponse), nil
}

func (s *DepartmentService) loadOrCacheDepartmentTree(ctx context.Context, key string, id uint, depth int, includeEmployees bool) (*dto.DepartmentTreeResponse, error) {
	cached, found, err := s.readCachedDepartmentTree(
		ctx,
		key,
		id,
		depth,
		includeEmployees,
	)
	if err != nil {
		return s.loadDepartmentTree(ctx, id, depth, includeEmployees)
	}
	if found {
		return cached, nil
	}

	tree, err := s.loadDepartmentTree(ctx, id, depth, includeEmployees)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(tree)
	if err != nil || len(data) > maxDepartmentTreeCacheJSONSize {
		return tree, nil
	}

	_ = s.treeCache.store.Set(ctx, key, data, s.treeCache.jitter(s.treeCache.ttl))

	return tree, nil
}

func (s *DepartmentService) readCachedDepartmentTree(ctx context.Context, key string, id uint, depth int, includeEmployees bool) (*dto.DepartmentTreeResponse, bool, error) {
	data, found, err := s.treeCache.store.Get(ctx, key)
	if err != nil || !found {
		return nil, false, err
	}

	var cached dto.DepartmentTreeResponse
	if err := json.Unmarshal(data, &cached); err != nil ||
		!validCachedDepartmentTree(&cached, id, depth, includeEmployees) {
		return nil, false, nil
	}

	return &cached, true, nil
}

func validCachedDepartmentTree(tree *dto.DepartmentTreeResponse, wantID uint, depth int, includeEmployees bool) bool {
	if tree.ID != wantID {
		return false
	}

	return validCachedDepartmentTreeNode(tree, depth, includeEmployees)
}

func validCachedDepartmentTreeNode(node *dto.DepartmentTreeResponse, remainingDepth int, includeEmployees bool) bool {
	if node.ID == 0 || node.Children == nil {
		return false
	}

	if includeEmployees {
		if node.Employees == nil || *node.Employees == nil {
			return false
		}
	} else if node.Employees != nil {
		return false
	}

	if remainingDepth <= 0 {
		return len(node.Children) == 0
	}

	for index := range node.Children {
		child := &node.Children[index]
		if child.ParentID == nil || *child.ParentID != node.ID {
			return false
		}
		if !validCachedDepartmentTreeNode(child, remainingDepth-1, includeEmployees) {
			return false
		}
	}

	return true
}

func readDepartmentTreeEpoch(ctx context.Context, store CacheStore) (int64, error) {
	data, found, err := store.Get(ctx, departmentTreeEpochKey)
	if err != nil {
		return 0, err
	}

	if !found {
		return 0, nil
	}

	epoch, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil || epoch < 0 {
		return 0, fmt.Errorf("invalid department tree cache epoch %q", data)
	}

	return epoch, nil
}

func departmentTreeCacheKey(epoch int64, id uint, depth int, includeEmployees bool) string {
	return fmt.Sprintf(
		"department-tree:v%d:id=%d:depth=%d:employees=%t",
		epoch,
		id,
		depth,
		includeEmployees,
	)
}

func bumpDepartmentTreeEpoch(ctx context.Context, store CacheStore) {
	if store == nil {
		return
	}

	_, _ = store.Increment(context.WithoutCancel(ctx), departmentTreeEpochKey)
}

func newTTLJitter(seed int64) func(time.Duration) time.Duration {
	random := rand.New(rand.NewSource(seed))
	var mutex sync.Mutex

	return func(base time.Duration) time.Duration {
		if base <= 0 {
			return base
		}

		maxDuration := time.Duration(1<<63 - 1)
		maxJitter := base / departmentTreeTTLJitterPercent
		if remaining := maxDuration - base; maxJitter > remaining {
			maxJitter = remaining
		}

		if maxJitter <= 0 {
			return base
		}

		mutex.Lock()
		jitter := time.Duration(random.Int63n(int64(maxJitter) + 1))
		mutex.Unlock()

		return base + jitter
	}
}
