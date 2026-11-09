package filters

import (
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func column(f Field) any {
	if f.Expression != nil {
		return f.Expression
	}
	return f.Column
}

// ApplyFilter adds a single grouped WHERE clause, an optional global search,
// and ordered, server-mapped columns to an existing GORM query.
func ApplyFilter(db *gorm.DB, model FilterRequestModel, fields FieldMap) (*gorm.DB, error) {
	criteria := make([]clause.Expression, 0, len(model.FilterCriteria))
	for _, c := range model.FilterCriteria {
		f, ok := fields.lookup(c.FieldName)
		if !ok {
			return nil, invalid("Unknown filter field.", map[string]any{
				"fieldName": c.FieldName,
			})
		}
		e, err := criterion(c, f)
		if err != nil {
			return nil, err
		}
		criteria = append(criteria, e)
	}
	if len(criteria) > 0 {
		g := model.FilterGroup
		if g == nil || len(g.Children) == 0 {
			g = andAll(len(criteria))
		}
		e, err := groupExpr(g, criteria)
		if err != nil {
			return nil, err
		}
		db = db.Where(e)
	}
	if search := strings.TrimSpace(model.SearchGlobalText); search != "" {
		parts := []clause.Expression{}
		for _, f := range fields {
			if f.Type == String {
				parts = append(parts, stringExpr(f, "Contains", search))
			}
		}
		if len(parts) == 0 {
			db = db.Where("1 = 0")
		} else {
			db = db.Where(join(parts, " OR "))
		}
	}
	if len(model.SortFields) > 0 {
		sorts := []string{}
		args := []any{}
		for _, s := range model.SortFields {
			f, ok := fields.lookup(s.FieldName)
			if !ok {
				return nil, invalid("Unknown sort field.", map[string]any{
					"fieldName": s.FieldName,
				})
			}
			direction := " DESC"
			if s.IsAscending {
				direction = " ASC"
			}
			sorts = append(sorts, "?"+direction)
			args = append(args, column(f))
		}
		db = db.Clauses(clause.OrderBy{
			Expression: clause.Expr{
				SQL:  strings.Join(sorts, ", "),
				Vars: args,
			},
		})
	}
	return db, nil
}

// Find executes the filtered count followed by the paginated query.
func Find[T any](db *gorm.DB, model FilterRequestModel, fields FieldMap) (FilterResponseModel[T], error) {
	result := FilterResponseModel[T]{
		Items:        []T{},
		CurrentPage:  model.CurrentPage,
		ItemsPerPage: model.ItemsPerPage,
	}
	if model.CurrentPage < 1 {
		return result, invalid("Current page must be a positive integer.", map[string]any{
			"field":       "currentPage",
			"currentPage": model.CurrentPage,
		})
	}
	if model.ItemsPerPage < 1 {
		return result, invalid("Items per page must be a positive integer.", map[string]any{
			"field":        "itemsPerPage",
			"itemsPerPage": model.ItemsPerPage,
		})
	}
	filtered, err := ApplyFilter(db, model, fields)
	if err != nil {
		return result, err
	}
	if err = filtered.Count(&result.TotalItems).Error; err != nil {
		return result, err
	}
	result.TotalPages = result.TotalItems / int64(model.ItemsPerPage)
	if result.TotalItems%int64(model.ItemsPerPage) != 0 {
		result.TotalPages++
	}
	result.HasPreviousPage = model.CurrentPage > 1
	result.HasNextPage = int64(model.CurrentPage) < result.TotalPages
	if int64(model.CurrentPage-1) > math.MaxInt64/int64(model.ItemsPerPage) {
		return result, invalid("Pagination offset is too large.", nil)
	}
	offset := int64(model.CurrentPage-1) * int64(model.ItemsPerPage)
	if offset > int64(math.MaxInt) {
		return result, invalid("Pagination offset is too large.", nil)
	}
	err = filtered.Limit(model.ItemsPerPage).Offset(int(offset)).Find(&result.Items).Error
	return result, err
}
func join(parts []clause.Expression, separator string) clause.Expression {
	ss := make([]string, len(parts))
	args := make([]any, len(parts))
	for i, p := range parts {
		ss[i] = "?"
		args[i] = p
	}
	return clause.Expr{
		SQL:  "(" + strings.Join(ss, separator) + ")",
		Vars: args,
	}
}
func groupExpr(g *Group, criteria []clause.Expression) (clause.Expression, error) {
	if g == nil || len(g.Children) == 0 {
		return nil, invalid("Filter group must contain at least one child.", nil)
	}
	parts := []clause.Expression{}
	for _, n := range g.Children {
		if (n.CriterionIndex == nil) == (n.Group == nil) {
			return nil, invalid("Each filter group node must define either a criterion index or a nested group.", nil)
		}
		if n.Group != nil {
			e, err := groupExpr(n.Group, criteria)
			if err != nil {
				return nil, err
			}
			parts = append(parts, e)
		} else {
			index := *n.CriterionIndex
			if index < 0 || index >= len(criteria) {
				return nil, invalid("Filter group references is invalid with the criterion index.", map[string]any{
					"criterionIndex": index,
				})
			}
			parts = append(parts, criteria[index])
		}
	}
	switch strings.ToUpper(g.Logic) {
	case "", "AND":
		return join(parts, " AND "), nil
	case "OR":
		return join(parts, " OR "), nil
	case "NOT":
		if len(parts) != 1 {
			return nil, invalid("NOT filter groups must contain exactly one child.", nil)
		}
		return clause.Expr{
			SQL: "NOT (?)",
			Vars: []any{
				parts[0],
			},
		}, nil
	default:
		return nil, invalid("Unsupported filter group logic.", map[string]any{
			"logic": g.Logic,
		})
	}
}
func criterion(c Criterion, f Field) (clause.Expression, error) {
	values := []string{}
	for _, v := range c.Values {
		if v = strings.TrimSpace(v); v != "" {
			values = append(values, v)
		}
	}
	if c.Operator == "IsNull" {
		if len(values) != 0 {
			return nil, invalid("Filter operator does not accept values.", map[string]any{
				"fieldName":  c.FieldName,
				"operator":   c.Operator,
				"valueCount": len(values),
			})
		}
		if !f.Nullable && f.Type != String {
			return nil, invalid("This operator is only supported for nullable fields.", map[string]any{
				"fieldName": c.FieldName,
				"operator":  c.Operator,
			})
		}
		return clause.Expr{
			SQL: "? IS NULL",
			Vars: []any{
				column(f),
			},
		}, nil
	}
	if len(values) == 0 {
		return nil, invalid("Filter field must contain at least one value.", map[string]any{
			"fieldName": c.FieldName,
		})
	}
	if (c.Operator == "GreaterThan" || c.Operator == "GreaterThanOrEquals" || c.Operator == "LessThan" ||
		c.Operator == "LessThanOrEquals") && len(values) != 1 {
		return nil, invalid("Filter operator requires exactly one value.", map[string]any{
			"fieldName":  c.FieldName,
			"operator":   c.Operator,
			"valueCount": len(values),
		})
	}
	if f.Type == Boolean && c.Operator != "Equals" {
		return nil, invalid("Boolean fields only support the Equal operator.", map[string]any{
			"fieldName": c.FieldName,
			"operator":  c.Operator,
		})
	}
	parts := []clause.Expression{}
	for _, raw := range values {
		expr := column(f)
		value, err := parseValue(raw, f, c.FieldName)
		if c.DateTimeFilterOptions != nil && c.DateTimeFilterOptions.TemporalPartType != "None" {
			expr, value, err = temporalExpr(f, raw, c.DateTimeFilterOptions.TemporalPartType)
		}
		if err != nil {
			return nil, err
		}
		switch c.Operator {
		case "Contains", "StartsWith", "EndsWith":
			if f.Type != String {
				return nil, invalid("This operator is only supported for string fields.", map[string]any{
					"fieldType": string(f.Type),
					"operator":  c.Operator,
				})
			}
			parts = append(parts, stringExpr(f, c.Operator, value.(string)))
		case "Equals", "GreaterThan", "GreaterThanOrEquals", "LessThan", "LessThanOrEquals":
			ops := map[string]string{
				"Equals":              "=",
				"GreaterThan":         ">",
				"GreaterThanOrEquals": ">=",
				"LessThan":            "<",
				"LessThanOrEquals":    "<=",
			}
			parts = append(parts, clause.Expr{
				SQL: "(? IS NOT NULL AND ? " + ops[c.Operator] + " ?)",
				Vars: []any{
					column(f),
					expr,
					value,
				},
			})
		default:
			return nil, invalid("Unsupported filter operator.", map[string]any{
				"fieldName": c.FieldName,
				"operator":  c.Operator,
			})
		}
	}
	return join(parts, " OR "), nil
}
func stringExpr(f Field, op, value string) clause.Expression {
	value = strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(value)
	switch op {
	case "Contains":
		value = "%" + value + "%"
	case "StartsWith":
		value += "%"
	case "EndsWith":
		value = "%" + value
	}
	return clause.Expr{
		SQL: "(? IS NOT NULL AND ? LIKE ? ESCAPE '\\')",
		Vars: []any{
			column(f),
			column(f),
			value,
		},
	}
}

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func parseValue(raw string, f Field, name string) (any, error) {
	bad := func() (any, error) {
		return nil, invalid("Invalid filter value for field.", map[string]any{
			"fieldName": name,
			"value":     raw,
		})
	}
	switch f.Type {
	case String:
		return raw, nil
	case UUID:
		if !uuidPattern.MatchString(raw) {
			return bad()
		}
		return raw, nil
	case Integer:
		n, e := strconv.ParseInt(raw, 10, 32)
		if e != nil {
			return bad()
		}
		return n, nil
	case Long:
		n, e := strconv.ParseInt(raw, 10, 64)
		if e != nil {
			return bad()
		}
		return n, nil
	case Decimal:
		if _, ok := new(big.Rat).SetString(raw); !ok {
			return bad()
		}
		return raw, nil
	case Float, Double:
		n, e := strconv.ParseFloat(raw, 64)
		if e != nil || math.IsInf(n, 0) || math.IsNaN(n) {
			return bad()
		}
		return n, nil
	case Boolean:
		if strings.EqualFold(raw, "true") {
			return true, nil
		}
		if strings.EqualFold(raw, "false") {
			return false, nil
		}
		return bad()
	case DateTime, DateTimeOffset, DateOnly:
		for _, layout := range []string{
			time.RFC3339Nano,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02",
		} {
			if v, e := time.Parse(layout, raw); e == nil {
				return v, nil
			}
		}
		return bad()
	case Enum:
		for k, v := range f.EnumValues {
			if strings.EqualFold(k, raw) {
				return v, nil
			}
		}
		return nil, invalid("Invalid filter value for enum.", map[string]any{
			"value": raw,
		})
	default:
		return nil, invalid("Unsupported filter value type for field.", map[string]any{
			"fieldName": name,
			"type":      f.Type,
		})
	}
}
func temporalExpr(f Field, raw, part string) (any, any, error) {
	if f.Type != DateOnly && f.Type != DateTime && f.Type != DateTimeOffset {
		return nil, nil, invalid("Temporal filters are not supported for this type.", map[string]any{
			"fieldType":    f.Type,
			"temporalPart": part,
		})
	}
	switch part {
	case "DateOnly":
		v, err := parseValue(raw, Field{
			Type: DateOnly,
		}, "")
		if err != nil {
			return nil, nil, err
		}
		return clause.Expr{
			SQL: "CAST(? AS DATE)",
			Vars: []any{
				column(f),
			},
		}, v.(time.Time).Format("2006-01-02"), nil
	case "Date", "Month", "Year":
		v, err := strconv.Atoi(raw)
		if err != nil {
			return nil, nil, invalid("Invalid filter value for field.", map[string]any{
				"value": raw,
			})
		}
		component := map[string]string{
			"Date":  "DAY",
			"Month": "MONTH",
			"Year":  "YEAR",
		}[part]
		return clause.Expr{
			SQL: fmt.Sprintf("EXTRACT(%s FROM ?)", component),
			Vars: []any{
				column(f),
			},
		}, v, nil
	default:
		return nil, nil, invalid("Unsupported temporal part type.", map[string]any{
			"temporalPart": part,
		})
	}
}
