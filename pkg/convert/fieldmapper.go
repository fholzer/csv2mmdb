package convert

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/pkg/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type FieldMapper interface {
	Map(string) (mmdbtype.DataType, error)
	ShouldOmitRecord(string) bool
	ShouldOmitValue(string) bool
	GetConfig() *FieldConfig
	GetTargetFieldComponents() []string
}

type BaseFieldMapper struct {
	targetFieldComponents []string
	caser                 *cases.Caser
	FieldConfig
}

func (m *BaseFieldMapper) ShouldOmitRecord(input string) bool {
	return m.Critical && input == ""
}

func (m *BaseFieldMapper) ShouldOmitValue(input string) bool {
	return m.IgnoreEmpty && input == ""
}

func (m *BaseFieldMapper) GetConfig() *FieldConfig {
	return &m.FieldConfig
}

func (m *BaseFieldMapper) GetTargetFieldComponents() []string {
	return m.targetFieldComponents
}

var titleCaser cases.Caser = cases.Title(language.English)
var upperCaser cases.Caser = cases.Upper(language.English)
var lowerCaser cases.Caser = cases.Lower(language.English)

func (m *BaseFieldMapper) Preprocess(s string) string {
	// capitalize
	if m.caser != nil {
		s = m.caser.String(s)
	}

	// translate
	// if m.Translate != nil {
	// 	if v, ok := m.Translate[s]; ok {
	// 		s = v
	// 	} else {
	// 		fmt.Printf("No translation for '%s' value '%s' with target field '%s'\n", m.Name, s, m.Target)
	// 	}
	// }

	// create mmdbtype maker
	return s
}

func NewFieldMapper(f *FieldConfig) (FieldMapper, error) {
	switch f.Type {
	case "string":
		return NewStringFieldMapper(f), nil
	case "int32":
		return NewInt32FieldMapper(f), nil
	case "uint32":
		return NewUint32FieldMapper(f), nil
	case "int64":
		return NewInt64FieldMapper(f), nil
	case "uint64":
		return NewUint64FieldMapper(f), nil
	case "boolean":
		return NewBooleanFieldMapper(f), nil
	case "float32":
		return NewFloat32FieldMapper(f), nil
	default:
		return nil, fmt.Errorf("unknown field type '%s' for field '%s'", f.Type, f.Name)
	}
}

func newBaseFieldMapper(fc *FieldConfig) *BaseFieldMapper {
	targetFieldComponents := strings.Split(fc.Target, ".")
	var caser *cases.Caser

	switch fc.Capitalization {
	case "":
		break
	case "lower":
		caser = &lowerCaser
	case "upper":
		caser = &upperCaser
	case "title":
		caser = &titleCaser
	default:
		panic(fmt.Sprintf("unknown capitalization mode '%s' for fiel '%s'", fc.Capitalization, fc.Name))
	}

	return &BaseFieldMapper{
		targetFieldComponents: targetFieldComponents,
		caser:                 caser,
		FieldConfig:           *fc,
	}
}

type StringFieldMapper struct {
	translator map[string]mmdbtype.String
	BaseFieldMapper
}

func NewStringFieldMapper(fc *FieldConfig) *StringFieldMapper {
	var translator map[string]mmdbtype.String
	if fc.Translate != nil {
		translator = map[string]mmdbtype.String{}
		for k, v := range fc.Translate {
			translator[k] = mmdbtype.String(v)
		}
	}

	return &StringFieldMapper{
		translator:      translator,
		BaseFieldMapper: *newBaseFieldMapper(fc),
	}
}

func (m *StringFieldMapper) Map(input string) (mmdbtype.DataType, error) {
	res := m.Preprocess(input)

	if m.translator != nil {
		if v, ok := m.translator[res]; ok {
			return v, nil
		} else {
			fmt.Printf("No translation for '%s' value '%s' with target field '%s'\n", m.Name, input, m.Target)
		}
	}
	return mmdbtype.String(res), nil
}

type Int32FieldMapper struct {
	BaseFieldMapper
}

func NewInt32FieldMapper(fc *FieldConfig) *Int32FieldMapper {
	return &Int32FieldMapper{
		BaseFieldMapper: *newBaseFieldMapper(fc),
	}
}

func (m *Int32FieldMapper) Map(input string) (mmdbtype.DataType, error) {
	i, err := strconv.ParseInt(input, 10, 32)
	if err != nil {
		return nil, errors.Wrapf(err, "Error converting field '%s' to int32: '%s'", m.Name, input)
	}
	return mmdbtype.Uint32(i), nil
}

type Uint32FieldMapper struct {
	BaseFieldMapper
}

func NewUint32FieldMapper(fc *FieldConfig) *Uint32FieldMapper {
	return &Uint32FieldMapper{
		BaseFieldMapper: *newBaseFieldMapper(fc),
	}
}

func (m *Uint32FieldMapper) Map(input string) (mmdbtype.DataType, error) {
	i, err := strconv.ParseUint(input, 10, 32)
	if err != nil {
		return nil, errors.Wrapf(err, "Error converting field '%s' to int32: '%s'", m.Name, input)
	}
	return mmdbtype.Uint32(i), nil
}

type Int64FieldMapper struct {
	BaseFieldMapper
}

func NewInt64FieldMapper(fc *FieldConfig) *Int64FieldMapper {
	return &Int64FieldMapper{
		BaseFieldMapper: *newBaseFieldMapper(fc),
	}
}

func (m *Int64FieldMapper) Map(input string) (mmdbtype.DataType, error) {
	i, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "Error converting field '%s' to int32: '%s'", m.Name, input)
	}
	return mmdbtype.Uint32(i), nil
}

type Uint64FieldMapper struct {
	BaseFieldMapper
}

func NewUint64FieldMapper(fc *FieldConfig) *Uint64FieldMapper {
	return &Uint64FieldMapper{
		BaseFieldMapper: *newBaseFieldMapper(fc),
	}
}

func (m *Uint64FieldMapper) Map(input string) (mmdbtype.DataType, error) {
	i, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "Error converting field '%s' to int32: '%s'", m.Name, input)
	}
	return mmdbtype.Uint32(i), nil
}

type BooleanFieldMapper struct {
	BaseFieldMapper
}

func NewBooleanFieldMapper(fc *FieldConfig) *BooleanFieldMapper {
	return &BooleanFieldMapper{
		BaseFieldMapper: *newBaseFieldMapper(fc),
	}
}

func (m *BooleanFieldMapper) Map(input string) (mmdbtype.DataType, error) {
	var b bool
	var err error
	// TODO make this behavior optional
	if input == "" {
		b = false
	} else {
		b, err = strconv.ParseBool(input)
		if err != nil {
			return nil, errors.Wrapf(err, "Error converting field '%s' to bool: '%s'", m.Name, input)
		}
	}

	if !b && m.OmitZeroValue {
		return nil, nil
	}

	return mmdbtype.Bool(b), nil
}

type Float32FieldMapper struct {
	BaseFieldMapper
}

func NewFloat32FieldMapper(fc *FieldConfig) *Float32FieldMapper {
	return &Float32FieldMapper{
		BaseFieldMapper: *newBaseFieldMapper(fc),
	}
}

func (m *Float32FieldMapper) Map(input string) (mmdbtype.DataType, error) {
	v, err := strconv.ParseFloat(input, 32)
	if err != nil {
		return nil, errors.Wrapf(err, "Error converting field '%s' to float32: '%s'", m.Name, input)
	}

	if v == 0 && m.OmitZeroValue {
		return nil, nil
	}

	return mmdbtype.Float32(v), nil
}
