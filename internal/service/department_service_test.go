package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/OmNom69/org-structure-api/internal/dto"
	"github.com/OmNom69/org-structure-api/internal/model"
	"github.com/OmNom69/org-structure-api/internal/storage"
)

type departmentTreeDepartmentRepository struct {
	DepartmentRepository

	departmentsByID    map[uint]*model.Department
	getByIDErrors      map[uint]error
	childrenByParentID map[uint][]model.Department
	getChildrenErrors  map[uint]error

	getByIDCalls     []uint
	getChildrenCalls []uint
}

func (f *departmentTreeDepartmentRepository) GetByID(ctx context.Context, id uint) (*model.Department, error) {
	f.getByIDCalls = append(f.getByIDCalls, id)

	if err, ok := f.getByIDErrors[id]; ok {
		return nil, err
	}

	department, ok := f.departmentsByID[id]
	if !ok {
		return nil, storage.ErrNotFound
	}

	return department, nil
}

func (f *departmentTreeDepartmentRepository) GetChildren(ctx context.Context, parentID uint) ([]model.Department, error) {
	f.getChildrenCalls = append(f.getChildrenCalls, parentID)

	if err, ok := f.getChildrenErrors[parentID]; ok {
		return nil, err
	}

	return f.childrenByParentID[parentID], nil
}

type departmentTreeEmployeeRepository struct {
	EmployeeRepository

	employeesByDepartmentID map[uint][]model.Employee
	errorsByDepartmentID    map[uint]error

	getEmployeesCalls []uint
}

func (f *departmentTreeEmployeeRepository) ListByDepartmentID(ctx context.Context, departmentID uint) ([]model.Employee, error) {
	f.getEmployeesCalls = append(f.getEmployeesCalls, departmentID)

	if err, ok := f.errorsByDepartmentID[departmentID]; ok {
		return nil, err
	}

	return f.employeesByDepartmentID[departmentID], nil
}

func TestDepartmentService_GetDepartmentTree_Validation(t *testing.T) {
	tests := []struct {
		name    string
		id      uint
		depth   int
		wantErr error
	}{
		{
			name:    "invalid department id",
			id:      0,
			depth:   1,
			wantErr: ErrInvalidDepartmentID,
		},
		{
			name:    "zero depth",
			id:      1,
			depth:   0,
			wantErr: ErrInvalidDepth,
		},
		{
			name:    "depth above maximum",
			id:      1,
			depth:   6,
			wantErr: ErrInvalidDepth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			departmentRepo := &departmentTreeDepartmentRepository{}
			employeeRepo := &departmentTreeEmployeeRepository{}
			departmentService := NewDepartmentService(departmentRepo, employeeRepo, nil, 0)

			tree, err := departmentService.GetDepartmentTree(
				context.Background(),
				tt.id,
				tt.depth,
				true,
			)

			if tree != nil {
				t.Fatalf("expected nil tree, got %+v", tree)
			}

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}

			if len(departmentRepo.getByIDCalls) != 0 {
				t.Fatalf("expected no GetByID calls, got %v", departmentRepo.getByIDCalls)
			}

			if len(departmentRepo.getChildrenCalls) != 0 {
				t.Fatalf("expected no GetChildren calls, got %v", departmentRepo.getChildrenCalls)
			}

			if len(employeeRepo.getEmployeesCalls) != 0 {
				t.Fatalf(
					"expected no ListByDepartmentID calls, got %v",
					employeeRepo.getEmployeesCalls,
				)
			}
		})
	}
}

func TestDepartmentService_GetDepartmentTree_BuildsTreeAndPreservesRepositoryOrder(t *testing.T) {
	root, childrenByParentID := newDepartmentTreeFixture()
	departmentRepo := &departmentTreeDepartmentRepository{
		departmentsByID: map[uint]*model.Department{
			root.ID: root,
		},
		childrenByParentID: childrenByParentID,
	}
	employeeRepo := &departmentTreeEmployeeRepository{}
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, nil, 0)

	tree, err := departmentService.GetDepartmentTree(context.Background(), root.ID, 2, false)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertDepartmentTreeNode(t, tree, *root)

	if len(tree.Children) != 2 {
		t.Fatalf("expected root to have 2 children, got %d", len(tree.Children))
	}

	finance := childrenByParentID[root.ID][0]
	backend := childrenByParentID[root.ID][1]
	platform := childrenByParentID[backend.ID][0]

	assertDepartmentTreeNode(t, &tree.Children[0], finance)
	assertDepartmentTreeNode(t, &tree.Children[1], backend)

	if len(tree.Children[1].Children) != 1 {
		t.Fatalf(
			"expected Backend to have 1 child, got %d",
			len(tree.Children[1].Children),
		)
	}

	assertDepartmentTreeNode(t, &tree.Children[1].Children[0], platform)

	assertEmptyNonNilChildren(t, &tree.Children[0])
	assertEmptyNonNilChildren(t, &tree.Children[1].Children[0])

	assertUintCalls(t, "GetByID", departmentRepo.getByIDCalls, []uint{1})
	assertUintCalls(t, "GetChildren", departmentRepo.getChildrenCalls, []uint{1, 4, 2})
	assertUintCalls(t, "ListByDepartmentID", employeeRepo.getEmployeesCalls, nil)
}

func TestDepartmentService_GetDepartmentTree_DepthControlsDescendantEdges(t *testing.T) {
	tests := []struct {
		name                string
		depth               int
		wantBackendChildren []uint
		wantGetChildren     []uint
	}{
		{
			name:                "depth one includes immediate children only",
			depth:               1,
			wantBackendChildren: nil,
			wantGetChildren:     []uint{1},
		},
		{
			name:                "depth two includes grandchildren",
			depth:               2,
			wantBackendChildren: []uint{3},
			wantGetChildren:     []uint{1, 4, 2},
		},
		{
			name:                "depth five is valid",
			depth:               5,
			wantBackendChildren: []uint{3},
			wantGetChildren:     []uint{1, 4, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, childrenByParentID := newDepartmentTreeFixture()
			departmentRepo := &departmentTreeDepartmentRepository{
				departmentsByID: map[uint]*model.Department{
					root.ID: root,
				},
				childrenByParentID: childrenByParentID,
			}
			employeeRepo := &departmentTreeEmployeeRepository{}
			departmentService := NewDepartmentService(departmentRepo, employeeRepo, nil, 0)

			tree, err := departmentService.GetDepartmentTree(
				context.Background(),
				root.ID,
				tt.depth,
				false,
			)

			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			if len(tree.Children) != 2 {
				t.Fatalf("expected root to have 2 children, got %d", len(tree.Children))
			}

			if tree.Children[0].ID != 4 || tree.Children[1].ID != 2 {
				t.Fatalf(
					"expected children order [4 2], got [%d %d]",
					tree.Children[0].ID,
					tree.Children[1].ID,
				)
			}

			backendChildIDs := make([]uint, 0, len(tree.Children[1].Children))
			for _, child := range tree.Children[1].Children {
				backendChildIDs = append(backendChildIDs, child.ID)
			}

			assertUintCalls(
				t,
				"Backend child IDs",
				backendChildIDs,
				tt.wantBackendChildren,
			)
			assertEmptyNonNilChildren(t, &tree.Children[0])
			assertUintCalls(t, "GetByID", departmentRepo.getByIDCalls, []uint{1})
			assertUintCalls(
				t,
				"GetChildren",
				departmentRepo.getChildrenCalls,
				tt.wantGetChildren,
			)
			assertUintCalls(t, "ListByDepartmentID", employeeRepo.getEmployeesCalls, nil)
		})
	}
}

func TestDepartmentService_GetDepartmentTree_WithoutEmployeesLeavesEmployeesNil(t *testing.T) {
	root, childrenByParentID := newDepartmentTreeFixture()
	departmentRepo := &departmentTreeDepartmentRepository{
		departmentsByID: map[uint]*model.Department{
			root.ID: root,
		},
		childrenByParentID: childrenByParentID,
	}
	employeeRepo := &departmentTreeEmployeeRepository{}
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, nil, 0)

	tree, err := departmentService.GetDepartmentTree(context.Background(), root.ID, 2, false)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertEmployeesNil(t, tree)
	assertUintCalls(t, "ListByDepartmentID", employeeRepo.getEmployeesCalls, nil)

	if len(tree.Children) != 2 || len(tree.Children[1].Children) != 1 {
		t.Fatalf("expected complete department tree, got %+v", tree)
	}
}

func TestDepartmentService_GetDepartmentTree_WithEmployeesLoadsEveryIncludedDepartment(t *testing.T) {
	root, childrenByParentID := newDepartmentTreeFixture()
	departmentRepo := &departmentTreeDepartmentRepository{
		departmentsByID: map[uint]*model.Department{
			root.ID: root,
		},
		childrenByParentID: childrenByParentID,
	}
	employeeRepo := &departmentTreeEmployeeRepository{
		employeesByDepartmentID: map[uint][]model.Employee{
			1: {
				{ID: 101, DepartmentID: 1, FullName: "Root Employee", Position: "CEO"},
			},
			2: {
				{ID: 102, DepartmentID: 2, FullName: "Backend Employee", Position: "Developer"},
			},
			3: {},
			4: {
				{ID: 104, DepartmentID: 4, FullName: "Finance Employee", Position: "Accountant"},
			},
		},
	}
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, nil, 0)

	tree, err := departmentService.GetDepartmentTree(context.Background(), root.ID, 2, true)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(tree.Children) != 2 ||
		tree.Children[0].ID != 4 ||
		tree.Children[1].ID != 2 ||
		len(tree.Children[1].Children) != 1 ||
		tree.Children[1].Children[0].ID != 3 {
		t.Fatalf("expected Finance followed by Backend with nested Platform, got %+v", tree)
	}

	assertEmployeeIDs(t, "Root", tree.Employees, []uint{101})
	assertEmployeeIDs(t, "Finance", tree.Children[0].Employees, []uint{104})
	assertEmployeeIDs(t, "Backend", tree.Children[1].Employees, []uint{102})
	assertEmployeeIDs(t, "Platform", tree.Children[1].Children[0].Employees, nil)

	if *tree.Children[1].Children[0].Employees == nil {
		t.Fatal("expected Platform employees to be a non-nil empty slice")
	}

	assertUintCalls(t, "GetByID", departmentRepo.getByIDCalls, []uint{1})
	assertUintCalls(t, "GetChildren", departmentRepo.getChildrenCalls, []uint{1, 4, 2})
	assertUintCalls(
		t,
		"ListByDepartmentID",
		employeeRepo.getEmployeesCalls,
		[]uint{1, 4, 2, 3},
	)
}

func TestDepartmentService_GetDepartmentTree_WithEmployeesPreservesEmptyNonNilSlice(t *testing.T) {
	root := &model.Department{
		ID:        1,
		Name:      "Root",
		CreatedAt: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
	}
	departmentRepo := &departmentTreeDepartmentRepository{
		departmentsByID: map[uint]*model.Department{
			root.ID: root,
		},
		childrenByParentID: map[uint][]model.Department{
			root.ID: {},
		},
	}
	employeeRepo := &departmentTreeEmployeeRepository{
		employeesByDepartmentID: map[uint][]model.Employee{
			root.ID: {},
		},
	}
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, nil, 0)

	tree, err := departmentService.GetDepartmentTree(context.Background(), root.ID, 1, true)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if tree.Employees == nil {
		t.Fatal("expected Employees to be non-nil")
	}

	if *tree.Employees == nil {
		t.Fatal("expected Employees to point to a non-nil empty slice")
	}

	if len(*tree.Employees) != 0 {
		t.Fatalf("expected no employees, got %d", len(*tree.Employees))
	}

	assertEmptyNonNilChildren(t, tree)
	assertUintCalls(t, "GetByID", departmentRepo.getByIDCalls, []uint{1})
	assertUintCalls(t, "GetChildren", departmentRepo.getChildrenCalls, []uint{1})
	assertUintCalls(t, "ListByDepartmentID", employeeRepo.getEmployeesCalls, []uint{1})
}

func TestDepartmentService_GetDepartmentTree_MapsDepartmentNotFound(t *testing.T) {
	departmentRepo := &departmentTreeDepartmentRepository{
		getByIDErrors: map[uint]error{
			7: storage.ErrNotFound,
		},
	}
	employeeRepo := &departmentTreeEmployeeRepository{}
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, nil, 0)

	tree, err := departmentService.GetDepartmentTree(context.Background(), 7, 1, true)

	if tree != nil {
		t.Fatalf("expected nil tree, got %+v", tree)
	}

	if !errors.Is(err, ErrDepartmentNotFound) {
		t.Fatalf("expected ErrDepartmentNotFound, got %v", err)
	}

	assertUintCalls(t, "GetByID", departmentRepo.getByIDCalls, []uint{7})
	assertUintCalls(t, "GetChildren", departmentRepo.getChildrenCalls, nil)
	assertUintCalls(t, "ListByDepartmentID", employeeRepo.getEmployeesCalls, nil)
}

func TestDepartmentService_GetDepartmentTree_PropagatesGetByIDError(t *testing.T) {
	getByIDErr := errors.New("database unavailable")
	departmentRepo := &departmentTreeDepartmentRepository{
		getByIDErrors: map[uint]error{
			9: getByIDErr,
		},
	}
	employeeRepo := &departmentTreeEmployeeRepository{}
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, nil, 0)

	tree, err := departmentService.GetDepartmentTree(context.Background(), 9, 1, true)

	if tree != nil {
		t.Fatalf("expected nil tree, got %+v", tree)
	}

	if !errors.Is(err, getByIDErr) {
		t.Fatalf("expected GetByID error, got %v", err)
	}

	assertUintCalls(t, "GetByID", departmentRepo.getByIDCalls, []uint{9})
	assertUintCalls(t, "GetChildren", departmentRepo.getChildrenCalls, nil)
	assertUintCalls(t, "ListByDepartmentID", employeeRepo.getEmployeesCalls, nil)
}

func TestDepartmentService_GetDepartmentTree_StopsAfterGetChildrenError(t *testing.T) {
	root, childrenByParentID := newDepartmentTreeFixture()
	childrenByParentID[root.ID] = []model.Department{
		childrenByParentID[root.ID][1],
		childrenByParentID[root.ID][0],
	}
	getChildrenErr := errors.New("get Backend children failed")
	departmentRepo := &departmentTreeDepartmentRepository{
		departmentsByID: map[uint]*model.Department{
			root.ID: root,
		},
		childrenByParentID: childrenByParentID,
		getChildrenErrors: map[uint]error{
			2: getChildrenErr,
		},
	}
	employeeRepo := &departmentTreeEmployeeRepository{}
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, nil, 0)

	tree, err := departmentService.GetDepartmentTree(context.Background(), root.ID, 2, false)

	if tree != nil {
		t.Fatalf("expected nil tree, got %+v", tree)
	}

	if !errors.Is(err, getChildrenErr) {
		t.Fatalf("expected GetChildren error, got %v", err)
	}

	assertUintCalls(t, "GetByID", departmentRepo.getByIDCalls, []uint{1})
	assertUintCalls(t, "GetChildren", departmentRepo.getChildrenCalls, []uint{1, 2})
	assertUintCalls(t, "ListByDepartmentID", employeeRepo.getEmployeesCalls, nil)
}

func TestDepartmentService_GetDepartmentTree_StopsAfterGetEmployeesError(t *testing.T) {
	root, childrenByParentID := newDepartmentTreeFixture()
	childrenByParentID[root.ID] = []model.Department{
		childrenByParentID[root.ID][1],
		childrenByParentID[root.ID][0],
	}
	getEmployeesErr := errors.New("get Backend employees failed")
	departmentRepo := &departmentTreeDepartmentRepository{
		departmentsByID: map[uint]*model.Department{
			root.ID: root,
		},
		childrenByParentID: childrenByParentID,
	}
	employeeRepo := &departmentTreeEmployeeRepository{
		employeesByDepartmentID: map[uint][]model.Employee{
			1: {},
		},
		errorsByDepartmentID: map[uint]error{
			2: getEmployeesErr,
		},
	}
	departmentService := NewDepartmentService(departmentRepo, employeeRepo, nil, 0)

	tree, err := departmentService.GetDepartmentTree(context.Background(), root.ID, 2, true)

	if tree != nil {
		t.Fatalf("expected nil tree, got %+v", tree)
	}

	if !errors.Is(err, getEmployeesErr) {
		t.Fatalf("expected ListByDepartmentID error, got %v", err)
	}

	assertUintCalls(t, "GetByID", departmentRepo.getByIDCalls, []uint{1})
	assertUintCalls(t, "GetChildren", departmentRepo.getChildrenCalls, []uint{1})
	assertUintCalls(t, "ListByDepartmentID", employeeRepo.getEmployeesCalls, []uint{1, 2})
}

func newDepartmentTreeFixture() (*model.Department, map[uint][]model.Department) {
	rootID := uint(1)
	backendID := uint(2)

	root := &model.Department{
		ID:        rootID,
		Name:      "Root",
		CreatedAt: time.Date(2026, time.January, 1, 9, 0, 0, 0, time.UTC),
	}
	backend := model.Department{
		ID:        backendID,
		Name:      "Backend",
		ParentID:  uintPointer(rootID),
		CreatedAt: time.Date(2026, time.January, 2, 9, 0, 0, 0, time.UTC),
	}
	platform := model.Department{
		ID:        3,
		Name:      "Platform",
		ParentID:  uintPointer(backendID),
		CreatedAt: time.Date(2026, time.January, 3, 9, 0, 0, 0, time.UTC),
	}
	finance := model.Department{
		ID:        4,
		Name:      "Finance",
		ParentID:  uintPointer(rootID),
		CreatedAt: time.Date(2026, time.January, 4, 9, 0, 0, 0, time.UTC),
	}

	return root, map[uint][]model.Department{
		rootID:     {finance, backend},
		backendID:  {platform},
		finance.ID: {},
	}
}

func assertDepartmentTreeNode(t *testing.T, got *dto.DepartmentTreeResponse, want model.Department) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("expected department ID %d, got %d", want.ID, got.ID)
	}

	if got.Name != want.Name {
		t.Fatalf("expected department name %q, got %q", want.Name, got.Name)
	}

	assertParentID(t, got.ParentID, want.ParentID)

	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("expected CreatedAt %s, got %s", want.CreatedAt, got.CreatedAt)
	}

	if got.Children == nil {
		t.Fatalf("expected department %d Children to be non-nil", got.ID)
	}
}

func assertParentID(t *testing.T, got *uint, want *uint) {
	t.Helper()

	if got == nil && want == nil {
		return
	}

	if got == nil || want == nil {
		t.Fatalf("expected ParentID %v, got %v", want, got)
	}

	if *got != *want {
		t.Fatalf("expected ParentID %d, got %d", *want, *got)
	}
}

func assertEmptyNonNilChildren(t *testing.T, tree *dto.DepartmentTreeResponse) {
	t.Helper()

	if tree.Children == nil {
		t.Fatalf("expected department %d Children to be non-nil", tree.ID)
	}

	if len(tree.Children) != 0 {
		t.Fatalf("expected department %d to have no children, got %d", tree.ID, len(tree.Children))
	}
}

func assertEmployeesNil(t *testing.T, tree *dto.DepartmentTreeResponse) {
	t.Helper()

	if tree.Employees != nil {
		t.Fatalf("expected department %d Employees to be nil", tree.ID)
	}

	for i := range tree.Children {
		assertEmployeesNil(t, &tree.Children[i])
	}
}

func assertEmployeeIDs(t *testing.T, departmentName string, employees *[]model.Employee, wantIDs []uint) {
	t.Helper()

	if employees == nil {
		t.Fatalf("expected %s Employees to be non-nil", departmentName)
	}

	gotIDs := make([]uint, 0, len(*employees))
	for _, employee := range *employees {
		gotIDs = append(gotIDs, employee.ID)
	}

	assertUintCalls(t, departmentName+" employee IDs", gotIDs, wantIDs)
}

func assertUintCalls(t *testing.T, method string, got []uint, want []uint) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %s calls with IDs %v, got %v", method, want, got)
	}

	remaining := make(map[uint]int, len(want))
	for _, id := range want {
		remaining[id]++
	}

	for _, id := range got {
		if remaining[id] == 0 {
			t.Fatalf("expected %s calls with IDs %v, got %v", method, want, got)
		}

		remaining[id]--
	}
}

func uintPointer(value uint) *uint {
	return &value
}
