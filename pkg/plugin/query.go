package plugin

import (
	"fmt"
	"strings"

	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
)

const (
	maxQueryTextLength       = 1024
	maxQueryFields           = 8
	maxQueryPointCount       = 10000
	maxHistoricalRangeSecs   = 10 * 366 * 24 * 60 * 60
	queryValidationErrorCode = "QUERY_VALIDATION_ERROR"
)

type PluginQuery struct {
	Type                PluginQueryType `json:"type"`
	EndpointID          string          `json:"endpoint"`
	Parameter           string          `json:"parameter"`
	Command             string          `json:"command"`
	From                int             `json:"from"`
	To                  int             `json:"to"`
	Fields              []string        `json:"fields"`
	Realtime            bool            `json:"realtime"`
	MaxPoints           int             `json:"points"`
	FrontendShiftedTime bool            `json:"frontendShiftedTime,omitempty"`

	// user-chosen split time from Grafana
	SplitAt int `json:"splitAt,omitempty"`

	// YAMCS parameter filter configuration
	YamcsFilter *YamcsFilterConfig `json:"yamcsFilter,omitempty"`
}

// YamcsFilterConfig defines client-side YAMCS parameter filtering
type YamcsFilterConfig struct {
	Enabled   bool   `json:"enabled"`
	Parameter string `json:"parameter"` // Name of parameter to filter by
	Operator  string `json:"operator"`  // "equals" only for now
	Value     string `json:"value"`     // Expected value for comparison
}

func (q PluginQuery) Validate() error {
	if err := validateRequiredText("endpoint", q.EndpointID); err != nil {
		return err
	}
	if q.MaxPoints != 0 {
		if err := validatePointCount(q.MaxPoints); err != nil {
			return err
		}
	}

	switch q.Type {
	case Graph:
		if err := q.validateParameterQuery(true, true); err != nil {
			return err
		}
		if len(q.Fields) > maxQueryFields {
			return validationError("too many query fields")
		}
		for _, field := range q.Fields {
			if field != "min" && field != "max" {
				return validationError(fmt.Sprintf("invalid query field %q", field))
			}
		}
	case DiscreteValue:
		if err := q.validateParameterQuery(true, true); err != nil {
			return err
		}
	case SingleValue, Image:
		if err := q.validateParameterQuery(false, false); err != nil {
			return err
		}
	case Events, CommandHistory:
		if err := q.validateTimeRange(); err != nil {
			return err
		}
	case Commanding:
		if err := validateRequiredText("command", q.Command); err != nil {
			return err
		}
	case Time, Alarms, Links, Demands, Subscriptions:
	default:
		return exception.New("query type not identified", "QUERY_TYPE_NOT_FOUND")
	}

	if q.YamcsFilter != nil {
		return q.YamcsFilter.Validate()
	}

	return nil
}

func (q PluginQuery) validateParameterQuery(requireTimeRange bool, requirePoints bool) error {
	if err := validateRequiredText("parameter", q.Parameter); err != nil {
		return err
	}
	if requireTimeRange {
		if err := q.validateTimeRange(); err != nil {
			return err
		}
	}
	if requirePoints {
		return validatePointCount(q.MaxPoints)
	}
	return q.validateOptionalPointCount()
}

func (q PluginQuery) validateTimeRange() error {
	if q.From < 0 || q.To < 0 {
		return validationError("time range cannot be negative")
	}
	if q.To <= q.From {
		return validationError("query end time must be after start time")
	}
	if q.To-q.From > maxHistoricalRangeSecs {
		return validationError(fmt.Sprintf("query time range cannot exceed %d seconds", maxHistoricalRangeSecs))
	}
	if q.SplitAt != 0 && (q.SplitAt < q.From || q.SplitAt > q.To) {
		return validationError("split time must be within query time range")
	}
	return nil
}

func (q PluginQuery) validateOptionalPointCount() error {
	if q.MaxPoints == 0 {
		return nil
	}
	return validatePointCount(q.MaxPoints)
}

func validatePointCount(count int) error {
	if count <= 0 || count > maxQueryPointCount {
		return validationError(fmt.Sprintf("invalid point count %d, must be between 1 and %d", count, maxQueryPointCount))
	}
	return nil
}

func (filter YamcsFilterConfig) Validate() error {
	if !filter.Enabled {
		return nil
	}
	if err := validateRequiredText("filter parameter", filter.Parameter); err != nil {
		return err
	}
	if len(filter.Value) > maxQueryTextLength {
		return validationError("filter value is too long")
	}
	if filter.Operator != "" && filter.Operator != "equals" {
		return validationError(fmt.Sprintf("unsupported filter operator %q", filter.Operator))
	}
	return nil
}

func validateRequiredText(name string, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return validationError(name + " is required")
	}
	if len(trimmed) > maxQueryTextLength {
		return validationError(name + " is too long")
	}
	return nil
}

func validationError(message string) error {
	return exception.New(message, queryValidationErrorCode)
}

type PluginQueryType string

const (
	Graph          PluginQueryType = "plot"
	SingleValue    PluginQueryType = "single"
	DiscreteValue  PluginQueryType = "discrete"
	Events         PluginQueryType = "events"
	Time           PluginQueryType = "time"
	Image          PluginQueryType = "image"
	Commanding     PluginQueryType = "commanding"
	CommandHistory PluginQueryType = "command-history"
	Alarms         PluginQueryType = "alarms"
	Links          PluginQueryType = "links"
	Demands        PluginQueryType = "demands"
	Subscriptions  PluginQueryType = "subscriptions"
)
