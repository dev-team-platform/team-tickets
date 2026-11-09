package filters

import (
	"math"
	"strconv"
	"strings"
	"time"
)

type FilterRequest struct {
	CurrentPage      *int                     `json:"currentPage"`
	ItemsPerPage     *int                     `json:"itemsPerPage"`
	SearchGlobalText *string                  `json:"searchGlobalText"`
	GroupPattern     *string                  `json:"groupPattern"`
	FilterCriteria   []FilterCriterionRequest `json:"filterCriteria"`
	SortFields       []SortFieldRequest       `json:"sortFields"`
	TimeZoneID       *string                  `json:"timeZoneId"`
}
type FilterCriterionRequest struct {
	FieldName             string                        `json:"fieldName"`
	Operator              string                        `json:"operator"`
	Values                []string                      `json:"values"`
	DateTimeFilterOptions *DateTimeFilterOptionsRequest `json:"dateTimeFilterOptions"`
}
type DateTimeFilterOptionsRequest struct {
	OffsetMinutes    int    `json:"offsetMinutes"`
	TemporalPartType string `json:"temporalPartType"`
}
type SortFieldRequest struct {
	FieldName   string `json:"fieldName"`
	IsAscending bool   `json:"isAscending"`
}
type FilterResponse[T any] struct {
	Items           []T   `json:"items"`
	CurrentPage     int   `json:"currentPage"`
	ItemsPerPage    int   `json:"itemsPerPage"`
	TotalItems      int64 `json:"totalItems"`
	TotalPages      int64 `json:"totalPages"`
	HasPreviousPage bool  `json:"hasPreviousPage"`
	HasNextPage     bool  `json:"hasNextPage"`
}

func ToFilterRequestModel(r FilterRequest, fields FieldMap) (FilterRequestModel, error) {
	m := FilterRequestModel{
		CurrentPage:  1,
		ItemsPerPage: math.MaxInt32,
		TimeZoneID:   "UTC",
	}
	if r.CurrentPage != nil {
		m.CurrentPage = *r.CurrentPage
	}
	if m.CurrentPage < 1 {
		return m, invalid("Current page must be a positive integer.", map[string]any{
			"field": "currentPage",
		})
	}
	if r.ItemsPerPage != nil {
		m.ItemsPerPage = *r.ItemsPerPage
	}
	if m.ItemsPerPage < 1 {
		return m, invalid("Items per page must be a positive integer.", map[string]any{
			"field": "itemsPerPage",
		})
	}
	if r.SearchGlobalText != nil {
		m.SearchGlobalText = *r.SearchGlobalText
	}
	if r.TimeZoneID != nil {
		if _, err := time.LoadLocation(*r.TimeZoneID); err != nil || *r.TimeZoneID == "" {
			return m, invalid("Invalid timezone ID.", map[string]any{
				"timeZoneId": *r.TimeZoneID,
			})
		}
		m.TimeZoneID = *r.TimeZoneID
	}
	for _, c := range r.FilterCriteria {
		if _, ok := fields.lookup(c.FieldName); !ok {
			return m, invalid("Unknown filter field.", map[string]any{
				"fieldName": c.FieldName,
			})
		}
		op, err := parseEnum(c.Operator, []string{
			"Equals",
			"GreaterThan",
			"LessThan",
			"GreaterThanOrEquals",
			"LessThanOrEquals",
			"Contains",
			"StartsWith",
			"EndsWith",
			"IsNull",
		}, "FilterOperator")
		if err != nil {
			return m, err
		}
		var opts *DateTimeFilterOptionsModel
		if c.DateTimeFilterOptions != nil {
			copy := DateTimeFilterOptionsModel{
				OffsetMinutes: c.DateTimeFilterOptions.OffsetMinutes,
			}
			copy.TemporalPartType, err = parseEnum(c.DateTimeFilterOptions.TemporalPartType, []string{
				"None",
				"DateOnly",
				"Date",
				"Month",
				"Year",
			}, "TemporalPartType")
			if err != nil {
				return m, err
			}
			opts = &copy
		}
		m.FilterCriteria = append(m.FilterCriteria, Criterion{
			c.FieldName,
			op,
			c.Values,
			opts,
		})
	}
	for _, s := range r.SortFields {
		if _, ok := fields.lookup(s.FieldName); !ok {
			return m, invalid("Unknown sort field.", map[string]any{
				"fieldName": s.FieldName,
			})
		}
		m.SortFields = append(m.SortFields, SortField{
			s.FieldName,
			s.IsAscending,
		})
	}
	pattern := ""
	if r.GroupPattern != nil {
		pattern = *r.GroupPattern
	}
	var err error
	m.FilterGroup, err = BuildGroup(pattern, len(m.FilterCriteria))
	return m, err
}
func parseEnum(value string, choices []string, name string) (string, error) {
	for _, v := range choices {
		if strings.EqualFold(strings.TrimSpace(value), v) {
			return v, nil
		}
	}
	if index, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		if index >= 1 && index <= len(choices) {
			return choices[index-1], nil
		}
		return strconv.Itoa(index), nil
	}
	return "", invalid("This is not a valid value for the enum.", map[string]any{
		"enumType": name,
		"value":    value,
	})
}

// FromFilterResponseModel maps a query result to its API response.
func FromFilterResponseModel[T any](model FilterResponseModel[T]) FilterResponse[T] {
	return FilterResponse[T]{
		Items:           model.Items,
		CurrentPage:     model.CurrentPage,
		ItemsPerPage:    model.ItemsPerPage,
		TotalItems:      model.TotalItems,
		TotalPages:      model.TotalPages,
		HasPreviousPage: model.HasPreviousPage,
		HasNextPage:     model.HasNextPage,
	}
}
