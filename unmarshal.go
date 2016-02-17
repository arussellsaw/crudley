package rest

import (
	"encoding/json"
	"reflect"
)

const (
	structTagRest        = "rest"
	structFieldImmutable = "immutable"
)

// RestrictedModel wraps a Model to provide immutability of fields from users
type RestrictedModel struct {
	Model
}

// UnmarshalJSON implements the json.Unmarshaler interface, but ignores fields
// marked with the struct tag rest:"immutable"
func (r *RestrictedModel) UnmarshalJSON(buf []byte) error {
	m := r.Model
	err := json.Unmarshal(buf, m)
	if err != nil {
		return err
	}
	defaults := getDefaults(reflect.ValueOf(m).Elem())
	setDefaults(reflect.ValueOf(m).Elem(), defaults)
	return nil
}

func getDefaults(val reflect.Value) map[int]interface{} {
	var defaults = make(map[int]interface{})
	valType := val.Type()
	for i := 0; i < val.NumField(); i++ {
		valFieldType := valType.Field(i)
		valFieldValue := val.Field(i)
		if valFieldType.Tag.Get(structTagRest) == structFieldImmutable {
			defaults[i] = valFieldValue.Interface()
		} else if valFieldValue.Kind() == reflect.Struct {
			defaults[i] = getDefaults(valFieldValue)
		}
	}
	return defaults
}

func setDefaults(val reflect.Value, defaults map[int]interface{}) {
	vt := val.Type()
	for i, fieldDefault := range defaults {
		fv := val.Field(i)
		ft := vt.Field(i)
		if ft.Tag.Get(structTagRest) == structFieldImmutable {
			if fv.CanSet() {
				fv.Set(reflect.ValueOf(fieldDefault))
			}
		} else if fv.Kind() == reflect.Struct {
			setDefaults(fv, defaults[i].(map[int]interface{}))
		}
	}
}
