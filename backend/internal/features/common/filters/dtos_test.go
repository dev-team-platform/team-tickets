package filters

import (
	"encoding/json"
	"strings"
	"testing"

	"gorm.io/gorm/clause"
)

func TestToFilterRequestModelMapsTemporalOptions(t *testing.T) {
	fields := FieldMap{
		"created": {
			Column: clause.Column{
				Name: "created",
			},
			Type: DateTime,
		},
	}
	request := FilterRequest{
		FilterCriteria: []FilterCriterionRequest{
			{
				FieldName: "created",
				Operator:  "equals",
				Values: []string{
					"2026",
				},
				DateTimeFilterOptions: &DateTimeFilterOptionsRequest{
					OffsetMinutes:    420,
					TemporalPartType: "year",
				},
			},
		},
	}
	model, err := ToFilterRequestModel(request, fields)
	if err != nil {
		t.Fatal(err)
	}
	options := model.FilterCriteria[0].DateTimeFilterOptions
	if model.FilterCriteria[0].Operator != "Equals" || options == nil || options.OffsetMinutes != 420 || options.TemporalPartType != "Year" {
		t.Fatalf("unexpected model: %+v", model)
	}
}

func TestFromFilterResponseModelMapsPagination(t *testing.T) {
	model := FilterResponseModel[string]{
		Items: []string{
			"item",
		},
		CurrentPage:     2,
		ItemsPerPage:    1,
		TotalItems:      3,
		TotalPages:      3,
		HasPreviousPage: true,
		HasNextPage:     true,
	}
	response := FromFilterResponseModel(model)
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"items":["item"],"currentPage":2,"itemsPerPage":1,"totalItems":3,"totalPages":3,"hasPreviousPage":true,"hasNextPage":true}`
	if strings.TrimSpace(string(data)) != want {
		t.Fatalf("unexpected response: %s", data)
	}
}
