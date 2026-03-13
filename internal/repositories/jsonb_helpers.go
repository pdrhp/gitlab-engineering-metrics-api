package repositories

import (
	"encoding/json"
	"fmt"
)

func assigneeContainsCondition(argIdx int) string {
	return fmt.Sprintf("((jsonb_typeof(assignees) = 'array' AND assignees ? $%d) OR (jsonb_typeof(assignees) = 'object' AND assignees ? 'current' AND (assignees->'current') ? $%d))", argIdx, argIdx)
}

func normalizedAssigneesJSONBExpr(column string) string {
	return fmt.Sprintf("CASE WHEN jsonb_typeof(%s) = 'array' THEN %s WHEN jsonb_typeof(%s) = 'object' AND %s ? 'current' THEN %s->'current' ELSE '[]'::jsonb END", column, column, column, column, column)
}

func decodeJSONBStringSlice(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}

	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}

	var obj struct {
		Current []string `json:"current"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}

	if obj.Current == nil {
		return []string{}, nil
	}

	return obj.Current, nil
}
