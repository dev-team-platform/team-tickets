// Package filters provides the shared dynamic filter request and GORM pipeline.
package filters

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	appErrors "github.com/trannhanlv2004/team-tickets/internal/features/common/errors"
	"gorm.io/gorm/clause"
)

type FilterRequestModel struct {
	CurrentPage      int
	ItemsPerPage     int
	SearchGlobalText string
	FilterGroup      *Group
	FilterCriteria   []Criterion
	SortFields       []SortField
	TimeZoneID       string
}
type Criterion struct {
	FieldName             string
	Operator              string
	Values                []string
	DateTimeFilterOptions *DateTimeFilterOptionsModel
}
type SortField struct {
	FieldName   string
	IsAscending bool
}
type DateTimeFilterOptionsModel struct {
	OffsetMinutes    int
	TemporalPartType string
}

type FilterResponseModel[T any] struct {
	Items           []T
	CurrentPage     int
	ItemsPerPage    int
	TotalItems      int64
	TotalPages      int64
	HasPreviousPage bool
	HasNextPage     bool
}

// Field is a trusted, server-defined database column and its filter type.
// Use Table to select a column from a joined relation; never populate it from a request.
type Field struct {
	Column     clause.Column
	Expression clause.Expression
	Type       FieldType
	Nullable   bool
	EnumValues map[string]any
}
type FieldType string

const (
	String         FieldType = "string"
	Integer        FieldType = "int"
	Long           FieldType = "long"
	Decimal        FieldType = "decimal"
	Float          FieldType = "float"
	Double         FieldType = "double"
	Boolean        FieldType = "bool"
	UUID           FieldType = "uuid"
	DateTime       FieldType = "datetime"
	DateTimeOffset FieldType = "datetimeoffset"
	DateOnly       FieldType = "dateonly"
	Enum           FieldType = "enum"
)

type FieldMap map[string]Field

func (m FieldMap) lookup(name string) (Field, bool) {
	for k, v := range m {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(name)) && strings.TrimSpace(name) != "" {
			return v, true
		}
	}
	return Field{}, false
}
func invalid(message string, details map[string]any) error {
	return appErrors.NewUnprocessableEntity(message, details)
}

func (m FilterRequestModel) GetContentHash() (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
