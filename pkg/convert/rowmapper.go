package convert

import (
	"fmt"
	"strings"

	"github.com/maxmind/mmdbwriter/mmdbtype"
)

type (
	mmdbRow = mmdbtype.Map
	MapFunc func([]string) mmdbRow
)

type RowMapper struct {
	config                   *Config
	fieldConfigMapping       map[string]FieldMapper
	targetFields             map[string]*FieldConfig
	sourceFieldNames         []string
	sourceFieldHeaderOffsets map[string]int
	stringCache              map[string]mmdbtype.String
}

func NewMapper(config *Config, header []string) (*RowMapper, error) {
	if header[0] != STR_START_IP || header[1] != STR_END_IP {
		return nil, fmt.Errorf("expecting '%s' and '%s' to be the first two column headers. Found '%s' and '%s'", STR_START_IP, STR_END_IP, header[0], header[1])
	}

	sourceFieldHeaderOffsets := map[string]int{}
	var sourceFieldNames []string
	fieldConfigMapping := map[string]FieldMapper{}
	// stored the first FieldConfig that causes that object to be created
	targetFields := map[string]*FieldConfig{}
	targetToSourceOffsetMapping := map[string]int{}

	for _, fieldConfig := range config.Fields {
		sourceFieldNames = append(sourceFieldNames, fieldConfig.Name)
		ft := fieldConfig.Target

		// find field's header offset
		foundHeader := false
		for i, v := range header {
			if v == fieldConfig.Name {
				sourceFieldHeaderOffsets[fieldConfig.Name] = i
				targetToSourceOffsetMapping[fieldConfig.Target] = i
				foundHeader = true
			}
		}
		if !foundHeader {
			return nil, fmt.Errorf("field '%s' for target '%s' not found in input file", fieldConfig.Name, fieldConfig.Target)
		}

		// check for duplicate targets
		if prevField, ok := fieldConfigMapping[ft]; ok {
			return nil, fmt.Errorf("duplicate target fields, field '%s' and '%s', both target '%s'", prevField.GetConfig().Name, fieldConfig.Name, ft)
		}

		var err error
		fieldConfigMapping[ft], err = NewFieldMapper(fieldConfig)
		if err != nil {
			return nil, err
		}

		// extract objects names from field paths and populate targetFields
		fieldComponents := strings.Split(fieldConfig.Target, ".")
		for i := 0; i < len(fieldComponents)-1; i++ {
			fc := strings.Join(fieldComponents[:i+1], ".")
			if _, ok := targetFields[fc]; !ok {
				targetFields[fc] = fieldConfig
			}
		}
	}

	// check for conflicts between object and value targets
	for fn, fc := range fieldConfigMapping {
		if objOriginConfig, ok := targetFields[fn]; ok && fc.GetConfig() != objOriginConfig {
			return nil, fmt.Errorf("target '%s' of field '%s' conflicts with object created by target of field '%s'", fc.GetConfig().Target, fc.GetConfig().Name, objOriginConfig.Name)
		}
	}

	return &RowMapper{
		config:                   config,
		fieldConfigMapping:       fieldConfigMapping,
		sourceFieldNames:         sourceFieldNames,
		targetFields:             targetFields,
		sourceFieldHeaderOffsets: sourceFieldHeaderOffsets,
		stringCache:              map[string]mmdbtype.String{},
	}, nil
}

func (m *RowMapper) Map(data []string) (mmdbRow, error) {
	r := mmdbRow{}

	for _, fieldConfig := range m.fieldConfigMapping {
		// prepare value
		val := data[m.sourceFieldHeaderOffsets[fieldConfig.GetConfig().Name]]

		if fieldConfig.ShouldOmitRecord(val) {
			return nil, nil
		}

		if fieldConfig.ShouldOmitValue(val) {
			continue
		}

		mmdbVal, err := fieldConfig.Map(val)
		if err != nil {
			return nil, err
		}

		if mmdbVal == nil {
			continue
		}

		// prepare location at which to store value
		components := fieldConfig.GetTargetFieldComponents()
		pathComponents := components[:len(components)-1]
		loc := r
		if len(pathComponents) > 0 {
			var err error
			loc, err = m.getMapByComponents(loc, pathComponents)
			if err != nil {
				return nil, err
			}
		}

		loc[mmdbtype.String(components[len(components)-1])] = mmdbVal
	}

	return r, nil
}

func (m *RowMapper) getCachedString(s string) mmdbtype.String {
	if cachedValue, ok := m.stringCache[s]; ok {
		return cachedValue
	}
	val := mmdbtype.String(s)
	m.stringCache[s] = val
	return val
}

func (m *RowMapper) getMapByComponents(parent mmdbRow, components []string) (mmdbRow, error) {
	this := m.getCachedString(components[0])
	if val, ok := parent[this]; ok {
		// check val type. Should be a map
		childMap, ok := val.(mmdbtype.Map)
		if !ok {
			return nil, fmt.Errorf("expected sub-field '%s' to be a map", this)
		}
		// in case there are components left...
		if len(components) > 1 {
			return m.getMapByComponents(childMap, components[1:])
		}
		// if not, we found the right map; return
		return childMap, nil
	} else {
		child := mmdbRow{}
		parent[this] = child

		// shortcut: if first doesn't exist, all others don't exist either,
		// create them here in one go
		for _, comp := range components[1:] {
			prev := child
			child = mmdbRow{}
			prev[m.getCachedString(comp)] = child
		}

		return child, nil
	}
}

// func (m *RowMapper) CanMergeRows(row1, row2 []string) bool {
// 	if row1 == nil || row2 == nil {
// 		return false
// 	}

// 	for _, fieldConfig := range m.fieldConfigMapping {
// 		// prepare value
// 		val1 := row1[m.sourceFieldHeaderOffsets[fieldConfig.GetConfig().Name]]
// 		val2 := row2[m.sourceFieldHeaderOffsets[fieldConfig.GetConfig().Name]]

// 		if val1 != val2 {
// 			return false
// 		}
// 	}
// 	return true
// }
