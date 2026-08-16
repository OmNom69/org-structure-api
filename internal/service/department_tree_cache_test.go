package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OmNom69/org-structure-api/internal/dto"
	"github.com/OmNom69/org-structure-api/internal/model"
	"github.com/OmNom69/org-structure-api/internal/storage"
)

type departmentTreeCacheSetCall struct {
	key   string
	value []byte
	ttl   time.Duration
}

type departmentTreeCacheFake struct {
	mu sync.Mutex

	values    map[string][]byte
	getErrors map[string]error
	setErr    error
	incrErr   error

	getKeys       []string
	setCalls      []departmentTreeCacheSetCall
	incrementKeys []string

	expectedEpochReads int
	epochReads         int
	allEpochReads      chan struct{}
	allEpochReadsOnce  sync.Once
}

func (f *departmentTreeCacheFake) Get(ctx context.Context, key string) ([]byte, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.getKeys = append(f.getKeys, key)
	if key == departmentTreeEpochKey {
		f.epochReads++
		if f.expectedEpochReads > 0 && f.epochReads == f.expectedEpochReads {
			f.allEpochReadsOnce.Do(func() { close(f.allEpochReads) })
		}
	}

	if err := f.getErrors[key]; err != nil {
		return nil, false, err
	}

	value, found := f.values[key]
	return append([]byte(nil), value...), found, nil
}

func (f *departmentTreeCacheFake) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.setCalls = append(f.setCalls, departmentTreeCacheSetCall{
		key:   key,
		value: append([]byte(nil), value...),
		ttl:   ttl,
	})
	if f.setErr != nil {
		return f.setErr
	}

	if f.values == nil {
		f.values = make(map[string][]byte)
	}
	f.values[key] = append([]byte(nil), value...)

	return nil
}

func (f *departmentTreeCacheFake) Increment(ctx context.Context, key string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.incrementKeys = append(f.incrementKeys, key)
	if f.incrErr != nil {
		return 0, f.incrErr
	}

	var current int64
	if value, found := f.values[key]; found {
		parsed, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil {
			return 0, err
		}
		current = parsed
	}

	current++
	if f.values == nil {
		f.values = make(map[string][]byte)
	}
	f.values[key] = []byte(strconv.FormatInt(current, 10))

	return current, nil
}

func (f *departmentTreeCacheFake) snapshotSets() []departmentTreeCacheSetCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	result := make([]departmentTreeCacheSetCall, len(f.setCalls))
	copy(result, f.setCalls)
	return result
}

func (f *departmentTreeCacheFake) snapshotIncrements() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.incrementKeys...)
}

type cacheTreeDepartmentRepository struct {
	DepartmentRepository

	mu sync.Mutex

	departments map[uint]model.Department
	children    map[uint][]model.Department

	getByIDCalls     []uint
	getChildrenCalls []uint
	createCalls      int

	getByIDStarted chan struct{}
	releaseGetByID chan struct{}
	startedOnce    sync.Once
}

func (f *cacheTreeDepartmentRepository) GetByID(ctx context.Context, id uint) (*model.Department, error) {
	f.mu.Lock()
	f.getByIDCalls = append(f.getByIDCalls, id)
	department, found := f.departments[id]
	f.mu.Unlock()

	if !found {
		return nil, storage.ErrNotFound
	}

	if f.getByIDStarted != nil {
		f.startedOnce.Do(func() { close(f.getByIDStarted) })
	}
	if f.releaseGetByID != nil {
		select {
		case <-f.releaseGetByID:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return &department, nil
}

func (f *cacheTreeDepartmentRepository) GetChildren(ctx context.Context, parentID uint) ([]model.Department, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.getChildrenCalls = append(f.getChildrenCalls, parentID)
	children := f.children[parentID]
	return append([]model.Department{}, children...), nil
}

func (f *cacheTreeDepartmentRepository) ExistsByNameAndParent(context.Context, string, *uint) (bool, error) {
	return false, nil
}

func (f *cacheTreeDepartmentRepository) Create(ctx context.Context, department *model.Department) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.createCalls++
	department.ID = 99
	return nil
}

func (f *cacheTreeDepartmentRepository) callCounts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.getByIDCalls), len(f.getChildrenCalls)
}

type cacheTreeEmployeeRepository struct {
	EmployeeRepository

	mu        sync.Mutex
	employees map[uint][]model.Employee
	calls     []uint
}

type cacheFilledBetweenChecksFake struct {
	payload        []byte
	payloadGets    int
	setCalls       int
	incrementCalls int
}

func (f *cacheFilledBetweenChecksFake) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if key == departmentTreeEpochKey {
		return []byte("3"), true, nil
	}

	f.payloadGets++
	if f.payloadGets == 1 {
		return nil, false, nil
	}

	return append([]byte(nil), f.payload...), true, nil
}

func (f *cacheFilledBetweenChecksFake) Set(context.Context, string, []byte, time.Duration) error {
	f.setCalls++
	return nil
}

func (f *cacheFilledBetweenChecksFake) Increment(context.Context, string) (int64, error) {
	f.incrementCalls++
	return 1, nil
}

func (f *cacheTreeEmployeeRepository) ListByDepartmentID(ctx context.Context, departmentID uint) ([]model.Employee, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, departmentID)
	employees := f.employees[departmentID]
	return append([]model.Employee{}, employees...), nil
}

func (f *cacheTreeEmployeeRepository) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.calls)
}

func TestDepartmentTreeCacheDisabledUsesRepository(t *testing.T) {
	departmentRepo, employeeRepo := newCacheTreeRepositories(1, "Root")
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, nil, 0)

	tree, err := departmentService.GetDepartmentTree(context.Background(), 1, 1, false)
	if err != nil {
		t.Fatalf("GetDepartmentTree() error = %v", err)
	}
	if tree.ID != 1 {
		t.Fatalf("GetDepartmentTree() ID = %d, want 1", tree.ID)
	}

	getByIDCalls, getChildrenCalls := departmentRepo.callCounts()
	if getByIDCalls != 1 || getChildrenCalls != 1 {
		t.Fatalf("repository calls = GetByID:%d GetChildren:%d, want 1 and 1", getByIDCalls, getChildrenCalls)
	}
}

func TestDepartmentTreeCacheMissLoadsRepositoryAndSetsCache(t *testing.T) {
	departmentRepo, employeeRepo := newCacheTreeRepositories(1, "Root")
	cacheStore := newDepartmentTreeCacheFake()
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, cacheStore, time.Minute)
	departmentService.treeCache.jitter = func(ttl time.Duration) time.Duration { return ttl }

	tree, err := departmentService.GetDepartmentTree(context.Background(), 1, 1, false)
	if err != nil {
		t.Fatalf("GetDepartmentTree() error = %v", err)
	}
	if tree.Name != "Root" {
		t.Fatalf("GetDepartmentTree() name = %q, want Root", tree.Name)
	}

	wantKey := "department-tree:v0:id=1:depth=1:employees=false"
	sets := cacheStore.snapshotSets()
	if len(sets) != 1 || sets[0].key != wantKey {
		t.Fatalf("cache SET calls = %+v, want one call for %q", sets, wantKey)
	}
	if sets[0].ttl != time.Minute {
		t.Fatalf("cache SET TTL = %s, want 1m", sets[0].ttl)
	}

	var cached dto.DepartmentTreeResponse
	if err := json.Unmarshal(sets[0].value, &cached); err != nil {
		t.Fatalf("cached JSON is invalid: %v", err)
	}
	if cached.ID != tree.ID || cached.Name != tree.Name {
		t.Fatalf("cached tree = %+v, response = %+v", cached, tree)
	}
}

func TestDepartmentTreeCacheHitSkipsRepositories(t *testing.T) {
	cacheStore := newDepartmentTreeCacheFake()
	cacheStore.values[departmentTreeEpochKey] = []byte("7")
	key := departmentTreeCacheKey(7, 5, 2, true)
	cacheStore.values[key] = mustMarshalTree(t, dto.DepartmentTreeResponse{
		ID:        5,
		Name:      "Cached",
		Employees: &[]model.Employee{},
		Children:  []dto.DepartmentTreeResponse{},
	})
	departmentRepo, employeeRepo := newCacheTreeRepositories(5, "Database")
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, cacheStore, time.Minute)

	tree, err := departmentService.GetDepartmentTree(context.Background(), 5, 2, true)
	if err != nil {
		t.Fatalf("GetDepartmentTree() error = %v", err)
	}
	if tree.Name != "Cached" {
		t.Fatalf("GetDepartmentTree() name = %q, want Cached", tree.Name)
	}

	getByIDCalls, getChildrenCalls := departmentRepo.callCounts()
	if getByIDCalls != 0 || getChildrenCalls != 0 || employeeRepo.callCount() != 0 {
		t.Fatalf(
			"repository calls = GetByID:%d GetChildren:%d Employees:%d, want all zero",
			getByIDCalls,
			getChildrenCalls,
			employeeRepo.callCount(),
		)
	}
}

func TestDepartmentTreeCacheDoubleCheckUsesValueFilledBeforeSingleflight(t *testing.T) {
	cacheStore := &cacheFilledBetweenChecksFake{
		payload: mustMarshalTree(t, dto.DepartmentTreeResponse{
			ID:       1,
			Name:     "Filled by another request",
			Children: []dto.DepartmentTreeResponse{},
		}),
	}
	departmentRepo, employeeRepo := newCacheTreeRepositories(1, "Database")
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, cacheStore, time.Minute)

	tree, err := departmentService.GetDepartmentTree(context.Background(), 1, 1, false)
	if err != nil {
		t.Fatalf("GetDepartmentTree() error = %v", err)
	}
	if tree.Name != "Filled by another request" {
		t.Fatalf("GetDepartmentTree() name = %q, want second cache value", tree.Name)
	}

	getByIDCalls, getChildrenCalls := departmentRepo.callCounts()
	if getByIDCalls != 0 || getChildrenCalls != 0 || employeeRepo.callCount() != 0 {
		t.Fatalf(
			"repository calls = GetByID:%d GetChildren:%d Employees:%d, want all zero",
			getByIDCalls,
			getChildrenCalls,
			employeeRepo.callCount(),
		)
	}
	if cacheStore.payloadGets != 2 {
		t.Fatalf("payload GET calls = %d, want outer miss plus inner hit", cacheStore.payloadGets)
	}
	if cacheStore.setCalls != 0 {
		t.Fatalf("cache SET calls = %d, want 0", cacheStore.setCalls)
	}
}

func TestDepartmentTreeCacheKeyIncludesAllResponseDimensions(t *testing.T) {
	tests := []struct {
		name             string
		epoch            int64
		id               uint
		depth            int
		includeEmployees bool
		want             string
	}{
		{"base", 12, 5, 2, true, "department-tree:v12:id=5:depth=2:employees=true"},
		{"epoch", 13, 5, 2, true, "department-tree:v13:id=5:depth=2:employees=true"},
		{"department ID", 12, 6, 2, true, "department-tree:v12:id=6:depth=2:employees=true"},
		{"depth", 12, 5, 3, true, "department-tree:v12:id=5:depth=3:employees=true"},
		{"employees", 12, 5, 2, false, "department-tree:v12:id=5:depth=2:employees=false"},
	}

	seen := make(map[string]struct{}, len(tests))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := departmentTreeCacheKey(tt.epoch, tt.id, tt.depth, tt.includeEmployees)
			if got != tt.want {
				t.Fatalf("departmentTreeCacheKey() = %q, want %q", got, tt.want)
			}
			if _, exists := seen[got]; exists {
				t.Fatalf("departmentTreeCacheKey() collision for %q", got)
			}
			seen[got] = struct{}{}
		})
	}
}

func TestDepartmentTreeCacheGetErrorFallsBackWithoutSet(t *testing.T) {
	cacheErr := errors.New("redis unavailable")
	cacheStore := newDepartmentTreeCacheFake()
	cacheStore.values[departmentTreeEpochKey] = []byte("2")
	key := departmentTreeCacheKey(2, 1, 1, false)
	cacheStore.getErrors[key] = cacheErr
	departmentRepo, employeeRepo := newCacheTreeRepositories(1, "Database")
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, cacheStore, time.Minute)

	tree, err := departmentService.GetDepartmentTree(context.Background(), 1, 1, false)
	if err != nil {
		t.Fatalf("GetDepartmentTree() error = %v", err)
	}
	if tree.Name != "Database" {
		t.Fatalf("GetDepartmentTree() name = %q, want Database", tree.Name)
	}
	if len(cacheStore.snapshotSets()) != 0 {
		t.Fatal("cache SET called after cache GET error")
	}
}

func TestDepartmentTreeEpochReadErrorAndMalformedValueBypassCache(t *testing.T) {
	tests := []struct {
		name       string
		epochValue []byte
		epochErr   error
	}{
		{name: "read error", epochErr: errors.New("read failed")},
		{name: "malformed", epochValue: []byte("not-an-integer")},
		{name: "negative", epochValue: []byte("-1")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheStore := newDepartmentTreeCacheFake()
			if tt.epochValue != nil {
				cacheStore.values[departmentTreeEpochKey] = tt.epochValue
			}
			if tt.epochErr != nil {
				cacheStore.getErrors[departmentTreeEpochKey] = tt.epochErr
			}
			departmentRepo, employeeRepo := newCacheTreeRepositories(1, "Database")
			departmentService := NewDepartmentService(departmentRepo, employeeRepo, cacheStore, time.Minute)

			tree, err := departmentService.GetDepartmentTree(context.Background(), 1, 1, false)
			if err != nil || tree == nil {
				t.Fatalf("GetDepartmentTree() = (%+v, %v), want successful fallback", tree, err)
			}
			if len(cacheStore.snapshotSets()) != 0 {
				t.Fatal("cache SET called without a valid epoch")
			}
		})
	}
}

func TestDepartmentTreeCacheSetErrorDoesNotFailResponse(t *testing.T) {
	cacheStore := newDepartmentTreeCacheFake()
	cacheStore.setErr = errors.New("write failed")
	departmentRepo, employeeRepo := newCacheTreeRepositories(1, "Database")
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, cacheStore, time.Minute)

	tree, err := departmentService.GetDepartmentTree(context.Background(), 1, 1, false)
	if err != nil || tree == nil {
		t.Fatalf("GetDepartmentTree() = (%+v, %v), want successful response", tree, err)
	}
	if len(cacheStore.snapshotSets()) != 1 {
		t.Fatalf("cache SET calls = %d, want 1", len(cacheStore.snapshotSets()))
	}
}

func TestDepartmentTreeCorruptCacheFallsBackAndRefills(t *testing.T) {
	tests := []struct {
		name             string
		includeEmployees bool
		value            []byte
	}{
		{name: "invalid JSON", value: []byte(`{"id":`)},
		{name: "JSON null", value: []byte(`null`)},
		{name: "empty object", value: []byte(`{}`)},
		{name: "wrong root ID", value: []byte(`{"id":2,"children":[]}`)},
		{name: "nil children", value: []byte(`{"id":1,"children":null}`)},
		{
			name:             "included employees are null",
			includeEmployees: true,
			value:            []byte(`{"id":1,"employees":null,"children":[]}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheStore := newDepartmentTreeCacheFake()
			cacheStore.values[departmentTreeEpochKey] = []byte("4")
			key := departmentTreeCacheKey(4, 1, 1, tt.includeEmployees)
			cacheStore.values[key] = tt.value
			departmentRepo, employeeRepo := newCacheTreeRepositories(1, "Database")
			departmentService := NewDepartmentService(
				departmentRepo,
				employeeRepo,
				cacheStore,
				time.Minute,
			)

			tree, err := departmentService.GetDepartmentTree(
				context.Background(),
				1,
				1,
				tt.includeEmployees,
			)
			if err != nil || tree == nil || tree.Name != "Database" {
				t.Fatalf("GetDepartmentTree() = (%+v, %v), want database response", tree, err)
			}

			sets := cacheStore.snapshotSets()
			if len(sets) != 1 || sets[0].key != key {
				t.Fatalf("cache refill calls = %+v, want key %q", sets, key)
			}
			var cached dto.DepartmentTreeResponse
			if err := json.Unmarshal(sets[0].value, &cached); err != nil || cached.Name != "Database" {
				t.Fatalf("refilled cache = (%+v, %v), want valid database response", cached, err)
			}
		})
	}
}

func TestDepartmentTreeCacheRoundTripPreservesNilAndEmptySemantics(t *testing.T) {
	tests := []struct {
		name             string
		includeEmployees bool
	}{
		{name: "employees omitted", includeEmployees: false},
		{name: "employees included but empty", includeEmployees: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheStore := newDepartmentTreeCacheFake()
			departmentRepo, employeeRepo := newCacheTreeRepositories(1, "Root")
			departmentService := NewDepartmentService(departmentRepo, employeeRepo, cacheStore, time.Minute)

			cold, err := departmentService.GetDepartmentTree(
				context.Background(),
				1,
				1,
				tt.includeEmployees,
			)
			if err != nil {
				t.Fatalf("cold GetDepartmentTree() error = %v", err)
			}
			warm, err := departmentService.GetDepartmentTree(
				context.Background(),
				1,
				1,
				tt.includeEmployees,
			)
			if err != nil {
				t.Fatalf("warm GetDepartmentTree() error = %v", err)
			}

			for phase, tree := range map[string]*dto.DepartmentTreeResponse{
				"cold": cold,
				"warm": warm,
			} {
				if tree.Children == nil || len(tree.Children) != 0 {
					t.Errorf("%s Children = %#v, want non-nil empty slice", phase, tree.Children)
				}
				if !tt.includeEmployees {
					if tree.Employees != nil {
						t.Errorf("%s Employees = %#v, want nil", phase, tree.Employees)
					}
					continue
				}
				if tree.Employees == nil || *tree.Employees == nil || len(*tree.Employees) != 0 {
					t.Errorf("%s Employees = %#v, want pointer to non-nil empty slice", phase, tree.Employees)
				}
			}

			getByIDCalls, _ := departmentRepo.callCounts()
			if getByIDCalls != 1 {
				t.Fatalf("GetByID calls after cold+warm = %d, want 1", getByIDCalls)
			}
		})
	}
}

func TestDepartmentTreeCacheTTLUsesDeterministicBoundedJitter(t *testing.T) {
	const seed = int64(42)
	baseTTL := 10 * time.Minute
	cacheStore := newDepartmentTreeCacheFake()
	departmentRepo, employeeRepo := newCacheTreeRepositories(1, "Root")
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, cacheStore, baseTTL)
	departmentService.treeCache.jitter = newTTLJitter(seed)
	wantTTL := newTTLJitter(seed)(baseTTL)

	if _, err := departmentService.GetDepartmentTree(context.Background(), 1, 1, false); err != nil {
		t.Fatalf("GetDepartmentTree() error = %v", err)
	}

	sets := cacheStore.snapshotSets()
	if len(sets) != 1 {
		t.Fatalf("cache SET calls = %d, want 1", len(sets))
	}
	if sets[0].ttl != wantTTL {
		t.Fatalf("cache SET TTL = %s, want deterministic %s", sets[0].ttl, wantTTL)
	}
	if sets[0].ttl < baseTTL || sets[0].ttl > baseTTL+baseTTL/10 {
		t.Fatalf("cache SET TTL = %s, want range [%s, %s]", sets[0].ttl, baseTTL, baseTTL+baseTTL/10)
	}
	if got := newTTLJitter(seed)(time.Nanosecond); got <= 0 {
		t.Fatalf("jittered positive TTL = %s, want > 0", got)
	}
}

func TestDepartmentTreeOversizedJSONSkipsCacheSet(t *testing.T) {
	departmentRepo, employeeRepo := newCacheTreeRepositories(
		1,
		strings.Repeat("x", maxDepartmentTreeCacheJSONSize+1),
	)
	cacheStore := newDepartmentTreeCacheFake()
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, cacheStore, time.Minute)

	tree, err := departmentService.GetDepartmentTree(context.Background(), 1, 1, false)
	if err != nil || tree == nil {
		t.Fatalf("GetDepartmentTree() = (%+v, %v), want successful response", tree, err)
	}
	if len(cacheStore.snapshotSets()) != 0 {
		t.Fatal("cache SET called for oversized JSON")
	}
}

func TestDepartmentTreeConcurrentMissesUseSingleflight(t *testing.T) {
	const requestCount = 100
	cacheStore := newDepartmentTreeCacheFake()
	cacheStore.expectedEpochReads = requestCount
	cacheStore.allEpochReads = make(chan struct{})
	departmentRepo, employeeRepo := newCacheTreeRepositories(1, "Root")
	departmentRepo.getByIDStarted = make(chan struct{})
	departmentRepo.releaseGetByID = make(chan struct{})
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, cacheStore, time.Minute)

	start := make(chan struct{})
	errorsChannel := make(chan error, requestCount)
	var waitGroup sync.WaitGroup
	for range requestCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			_, err := departmentService.GetDepartmentTree(context.Background(), 1, 1, false)
			errorsChannel <- err
		}()
	}
	close(start)

	waitForSignal(t, departmentRepo.getByIDStarted, "repository load to start")
	waitForSignal(t, cacheStore.allEpochReads, "all requests to read the epoch")
	close(departmentRepo.releaseGetByID)
	waitGroup.Wait()
	close(errorsChannel)

	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("concurrent GetDepartmentTree() error = %v", err)
		}
	}
	getByIDCalls, _ := departmentRepo.callCounts()
	if getByIDCalls != 1 {
		t.Fatalf("GetByID calls = %d, want 1", getByIDCalls)
	}
	if len(cacheStore.snapshotSets()) != 1 {
		t.Fatalf("cache SET calls = %d, want 1", len(cacheStore.snapshotSets()))
	}
}

func TestDepartmentTreeReadUsesCapturedEpochForCacheSet(t *testing.T) {
	cacheStore := newDepartmentTreeCacheFake()
	cacheStore.values[departmentTreeEpochKey] = []byte("10")
	departmentRepo, employeeRepo := newCacheTreeRepositories(1, "Old snapshot")
	departmentRepo.getByIDStarted = make(chan struct{})
	departmentRepo.releaseGetByID = make(chan struct{})
	reader := NewDepartmentService(departmentRepo, employeeRepo, cacheStore, time.Minute)
	reader.treeCache.jitter = func(ttl time.Duration) time.Duration { return ttl }

	resultChannel := make(chan *dto.DepartmentTreeResponse, 1)
	errorChannel := make(chan error, 1)
	go func() {
		tree, err := reader.GetDepartmentTree(context.Background(), 1, 1, false)
		resultChannel <- tree
		errorChannel <- err
	}()

	waitForSignal(t, departmentRepo.getByIDStarted, "old-epoch repository load to start")
	mutator := NewDepartmentService(departmentRepo, employeeRepo, cacheStore, time.Minute)
	if _, err := mutator.CreateDepartment(context.Background(), CreateDepartmentInput{Name: "New"}); err != nil {
		t.Fatalf("CreateDepartment() error = %v", err)
	}
	if got := cacheStore.snapshotIncrements(); len(got) != 1 || got[0] != departmentTreeEpochKey {
		t.Fatalf("epoch increments = %v, want [%s]", got, departmentTreeEpochKey)
	}

	close(departmentRepo.releaseGetByID)
	if err := <-errorChannel; err != nil {
		t.Fatalf("GetDepartmentTree() error = %v", err)
	}
	tree := <-resultChannel
	if tree == nil || tree.Name != "Old snapshot" {
		t.Fatalf("GetDepartmentTree() = %+v, want old snapshot", tree)
	}

	sets := cacheStore.snapshotSets()
	wantOldKey := departmentTreeCacheKey(10, 1, 1, false)
	newKey := departmentTreeCacheKey(11, 1, 1, false)
	if len(sets) != 1 || sets[0].key != wantOldKey {
		t.Fatalf("cache SET calls = %+v, want captured key %q", sets, wantOldKey)
	}
	if sets[0].key == newKey {
		t.Fatalf("old snapshot was stored under new epoch key %q", newKey)
	}
}

func newDepartmentTreeCacheFake() *departmentTreeCacheFake {
	return &departmentTreeCacheFake{
		values:    make(map[string][]byte),
		getErrors: make(map[string]error),
	}
}

func newCacheTreeRepositories(id uint, name string) (*cacheTreeDepartmentRepository, *cacheTreeEmployeeRepository) {
	return &cacheTreeDepartmentRepository{
			departments: map[uint]model.Department{
				id: {ID: id, Name: name, CreatedAt: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)},
			},
			children: make(map[uint][]model.Department),
		}, &cacheTreeEmployeeRepository{
			employees: make(map[uint][]model.Employee),
		}
}

func mustMarshalTree(t *testing.T, tree dto.DepartmentTreeResponse) []byte {
	t.Helper()

	data, err := json.Marshal(tree)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}

func waitForSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}
