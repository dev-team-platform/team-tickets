package filters

import (
	"errors"
	"strings"
	"testing"

	appErrors "github.com/trannhanlv2004/team-tickets/internal/features/common/errors"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type testPerson struct {
	ID     int
	Name   *string
	Age    int
	Active bool
}

func setup(t *testing.T) (*gorm.DB, FieldMap) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&testPerson{}); err != nil {
		t.Fatal(err)
	}
	name := func(s string) *string { return &s }
	rows := []testPerson{
		{
			1,
			name("Alice"),
			20,
			true,
		},
		{
			2,
			name("Bob"),
			30,
			false,
		},
		{
			3,
			nil,
			40,
			true,
		},
		{
			4,
			name("100% real"),
			20,
			true,
		},
		{
			5,
			name("Alicia"),
			30,
			true,
		},
	}
	if err = db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	fields := FieldMap{
		"name": {
			Column: clause.Column{
				Name: "name",
			},
			Type:     String,
			Nullable: true,
		},
		"age": {
			Column: clause.Column{
				Name: "age",
			},
			Type: Integer,
		},
		"active": {
			Column: clause.Column{
				Name: "active",
			},
			Type: Boolean,
		},
		"id": {
			Column: clause.Column{
				Name: "id",
			},
			Type: Integer,
		},
	}
	return db.Model(&testPerson{}), fields
}
func intPtr(n int) *int       { return &n }
func strPtr(s string) *string { return &s }
func TestFindFiltersGroupsSortsAndPages(t *testing.T) {
	db, fields := setup(t)
	r := FilterRequest{
		CurrentPage:  intPtr(1),
		ItemsPerPage: intPtr(1),
		GroupPattern: strPtr("{0} OR {1}"),
		FilterCriteria: []FilterCriterionRequest{
			{
				FieldName: " NAME ",
				Operator:  "StartsWith",
				Values: []string{
					"Ali",
				},
			},
			{
				FieldName: "name",
				Operator:  "Equals",
				Values: []string{
					"Bob",
				},
			},
			{
				FieldName: "active",
				Operator:  "Equals",
				Values: []string{
					"true",
				},
			},
		},
		SortFields: []SortFieldRequest{
			{
				FieldName:   "age",
				IsAscending: false,
			},
			{
				FieldName:   "id",
				IsAscending: true,
			},
		},
	}
	m, err := ToFilterRequestModel(r, fields)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Find[testPerson](db, m, fields)
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalItems != 2 || got.TotalPages != 2 || len(got.Items) != 1 || got.Items[0].ID != 5 || !got.HasNextPage || got.HasPreviousPage {
		t.Fatalf("unexpected result: %+v", got)
	}
}
func TestFindEscapesLikeAndPreservesNotNullSemantics(t *testing.T) {
	db, fields := setup(t)
	tests := []struct {
		pattern   string
		criterion FilterCriterionRequest
		want      []int
	}{
		{
			"",
			FilterCriterionRequest{
				FieldName: "name",
				Operator:  "Contains",
				Values: []string{
					"%",
				},
			},
			[]int{
				4,
			},
		},
		{
			"(NOT {0})",
			FilterCriterionRequest{
				FieldName: "name",
				Operator:  "Equals",
				Values: []string{
					"Alice",
				},
			},
			[]int{
				2,
				3,
				4,
				5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+strings.Join(tt.criterion.Values, ""), func(t *testing.T) {
			m, err := ToFilterRequestModel(FilterRequest{
				GroupPattern: &tt.pattern,
				FilterCriteria: []FilterCriterionRequest{
					tt.criterion,
				},
				SortFields: []SortFieldRequest{
					{
						FieldName:   "id",
						IsAscending: true,
					},
				},
			}, fields)
			if err != nil {
				t.Fatal(err)
			}
			res, err := Find[testPerson](db.Session(&gorm.Session{}), m, fields)
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Items) != len(tt.want) {
				t.Fatalf("got %+v, want %v", res.Items, tt.want)
			}
			for i, id := range tt.want {
				if res.Items[i].ID != id {
					t.Fatalf("got %+v, want %v", res.Items, tt.want)
				}
			}
		})
	}
}
func TestFilterGroupParser(t *testing.T) {
	g, err := BuildGroup("{0} OR {1} AND {2}", 4)
	if err != nil {
		t.Fatal(err)
	}
	if g.Logic != "AND" || g.Children[0].Group.Logic != "OR" || g.Children[0].Group.Children[1].Group.Logic != "AND" ||
		*g.Children[1].CriterionIndex != 3 {
		t.Fatalf("unexpected group %+v", g)
	}
	for _, s := range []string{
		"{2}",
		"{0} AND",
		"{0} {1}",
		"({0}",
		"{0} OR OR {1}",
		"NOT {0}",
	} {
		if _, err := BuildGroup(s, 2); err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
	if _, err := BuildGroup("({0} and (not {1}))", 2); err != nil {
		t.Fatal(err)
	}
}
func TestFilterRejectsInvalidValues(t *testing.T) {
	_, fields := setup(t)
	for _, criterion := range []FilterCriterionRequest{
		{
			FieldName: "unknown",
			Operator:  "Equals",
			Values: []string{
				"1",
			},
		},
		{
			FieldName: "age",
			Operator:  "GreaterThan",
			Values: []string{
				"1",
				"2",
			},
		},
		{
			FieldName: "active",
			Operator:  "Contains",
			Values: []string{
				"true",
			},
		},
		{
			FieldName: "age",
			Operator:  "Equals",
			Values: []string{
				"bad",
			},
		},
	} {
		r := FilterRequest{
			FilterCriteria: []FilterCriterionRequest{
				criterion,
			},
		}
		m, err := ToFilterRequestModel(r, fields)
		if err == nil {
			_, err = ApplyFilter(nil, m, fields)
		}
		var appErr *appErrors.Error
		if !errors.As(err, &appErr) || appErr.StatusCode != 422 {
			t.Fatalf("expected 422, got %v", err)
		}
	}
}

func TestGlobalSearchAndMultipleValues(t *testing.T) {
	db, fields := setup(t)
	request := FilterRequest{

		SearchGlobalText: strPtr("li"),

		FilterCriteria: []FilterCriterionRequest{
			{
				FieldName: "age",
				Operator:  "Equals",
				Values: []string{
					"20",
					"30",
				},
			},
		},

		SortFields: []SortFieldRequest{
			{
				FieldName:   "id",
				IsAscending: true,
			},
		},
	}
	model, err := ToFilterRequestModel(request, fields)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Find[testPerson](db, model, fields)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalItems != 2 || result.Items[0].ID != 1 || result.Items[1].ID != 5 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestPaginationDefaultsAndValidation(t *testing.T) {
	_, fields := setup(t)
	model, err := ToFilterRequestModel(FilterRequest{}, fields)
	if err != nil || model.CurrentPage != 1 || model.ItemsPerPage != 2147483647 {
		t.Fatalf("unexpected defaults: %+v, %v", model, err)
	}
	for _, request := range []FilterRequest{
		{
			CurrentPage: intPtr(0),
		},
		{
			ItemsPerPage: intPtr(0),
		},
	} {
		_, err := ToFilterRequestModel(request, fields)
		var appErr *appErrors.Error
		if !errors.As(err, &appErr) || appErr.StatusCode != 422 {
			t.Fatalf("expected 422, got %v", err)
		}
	}
}

func TestTemporalFilterUsesPostgresExpressions(t *testing.T) {
	db, _ := setup(t)
	connection, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	postgresDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: connection,
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	fields := FieldMap{
		"created": {
			Column: clause.Column{
				Name: "created_at",
			},
			Type:     DateTime,
			Nullable: true,
		},
	}
	request := FilterRequest{
		FilterCriteria: []FilterCriterionRequest{
			{
				FieldName: "created",
				Operator:  "Equals",
				Values: []string{
					"2025",
				},
				DateTimeFilterOptions: &DateTimeFilterOptionsRequest{
					TemporalPartType: "Year",
					OffsetMinutes:    420,
				},
			},
		},
	}
	model, err := ToFilterRequestModel(request, fields)
	if err != nil {
		t.Fatal(err)
	}
	query, err := ApplyFilter(postgresDB.Model(&testPerson{}), model, fields)
	if err != nil {
		t.Fatal(err)
	}
	statement := query.Find(&[]testPerson{}).Statement
	if !strings.Contains(statement.SQL.String(), "EXTRACT(YEAR FROM \"created_at\")") || len(statement.Vars) != 1 || statement.Vars[0] != 2025 {
		t.Fatalf("unexpected query: %s, %v", statement.SQL.String(), statement.Vars)
	}
}
