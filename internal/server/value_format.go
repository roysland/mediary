package server

import (
	"database/sql"
	"strconv"
)

type trackableValueFields struct {
	valueType string
	valueInt  sql.NullInt64
	valueBool sql.NullInt64
	valueText sql.NullString
	unit      sql.NullString
}

func formatTrackableValue(fields trackableValueFields) string {
	switch fields.valueType {
	case "integer":
		value := strconv.FormatInt(fields.valueInt.Int64, 10)
		if fields.unit.Valid {
			return value + " " + fields.unit.String
		}
		return value
	case "boolean":
		return "Yes"
	case "text":
		return fields.valueText.String
	default:
		if fields.valueText.Valid {
			return fields.valueText.String
		}
		if fields.valueInt.Valid {
			return strconv.FormatInt(fields.valueInt.Int64, 10)
		}
		if fields.valueBool.Valid {
			return "Yes"
		}
		return ""
	}
}
