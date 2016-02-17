package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorilla/context"
)

// Operators are for setting Query predicates
const (
	OperatorGreaterThan = "$gt"
	OperatorLessThan    = "$lt"
	OperatorNotEqual    = "$ne"
)

var (
	queryMap = map[string]string{
		"_before":      OperatorLessThan,
		"_lessthan":    OperatorLessThan,
		"_after":       OperatorGreaterThan,
		"_greaterthan": OperatorGreaterThan,
	}
)

// GetResponse retrieves the Response from the request's context
func GetResponse(r *http.Request) *Response {
	rs, found := context.GetOk(r, "response")
	if found {
		res, ok := rs.(*Response)
		if ok {
			return res
		}
		return nil
	}
	return nil
}

// UnmarshalQuery creates a Model from the query parameters of a http request.
func UnmarshalQuery(r *http.Request, m interface{}) error {
	mValue := reflect.ValueOf(m).Elem()
	for key, val := range r.URL.Query() {
		s := val[0]
		err := fillStructFields(key, s, mValue)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalMultiQuery unmarshals a request into multiple Models
func UnmarshalMultiQuery(r *http.Request, m Model) ([]Model, error) {
	var mdls []Model
	for key, val := range r.URL.Query() {
		for i, s := range val {
			if len(mdls) < i+1 {
				mdls = append(mdls, m.New(""))
			}
			mValue := reflect.ValueOf(mdls[i]).Elem()
			err := fillStructFields(key, s, mValue)
			if err != nil {
				return mdls, err
			}
		}
	}
	return mdls, nil
}

func fillStructFields(key string, val string, mValue reflect.Value) error {
	mType := mValue.Type()
	for i := 0; i < mValue.NumField(); i++ {
		mFieldType := mType.Field(i)
		mFieldValue := mValue.Field(i)
		if mFieldValue.Kind() == reflect.Struct {
			fillStructFields(key, val, mFieldValue)
		}
		jsonTag := mFieldType.Tag.Get("json")
		if strings.Split(jsonTag, ",")[0] == key {
			if mFieldValue.CanSet() {

				//Most of this switch statement is taken from encoding/json/decode.go
				switch mFieldValue.Kind() {
				default:
					if mFieldValue.Kind() == reflect.String {
						mFieldValue.SetString(val)
					} else {
						err := json.Unmarshal([]byte(fmt.Sprintf(`"%s"`, val)), mFieldValue.Addr().Interface())
						if err != nil {
							return err
						}
						return nil
					}
				case reflect.Interface:
					return fmt.Errorf("interface type Model fields are not supported")
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					n, err := strconv.ParseInt(val, 10, 64)
					if err != nil || mFieldValue.OverflowInt(n) {
						return fmt.Errorf("failed to parse int %s: %s", key, val)
					}
					mFieldValue.SetInt(n)
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
					n, err := strconv.ParseUint(val, 10, 64)
					if err != nil || mFieldValue.OverflowUint(n) {
						return fmt.Errorf("failed to parse uint %s: %s", key, val)
					}
					mFieldValue.SetUint(n)

				case reflect.Float32, reflect.Float64:
					n, err := strconv.ParseFloat(val, mFieldValue.Type().Bits())
					if err != nil || mFieldValue.OverflowFloat(n) {
						return fmt.Errorf("failed to parse float %s: %s", key, val)
					}
					mFieldValue.SetFloat(n)
				case reflect.Bool:
					switch strings.ToLower(val) {
					case "true", "1":
						mFieldValue.SetBool(true)
					case "false", "0":
						mFieldValue.SetBool(false)
					default:
						return fmt.Errorf("failed to parse bool, unrecognized value: %s", val)
					}
				case reflect.Ptr:
					switch mFieldType.Type.Elem().Kind() {
					default:
						if mFieldValue.Kind() == reflect.String {
							mFieldValue.SetString(val)
						} else {
							return fmt.Errorf("unknown query param pointer type")
						}
					case reflect.Bool:
						t := reflect.New(mFieldType.Type.Elem())
						switch strings.ToLower(val) {
						case "true", "1":
							t.Elem().SetBool(true)
						case "false", "0":
							t.Elem().SetBool(false)
						default:
							return fmt.Errorf("failed to parse bool, unrecognized value: %s", val)
						}
						mFieldValue.Set(t)
					case reflect.Interface:
						return fmt.Errorf("interface type Model fields are not supported")
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						t := reflect.New(mFieldType.Type.Elem())
						n, err := strconv.ParseInt(val, 10, 64)
						if err != nil || mFieldValue.OverflowInt(n) {
							return fmt.Errorf("failed to parse int %s: %s", key, val)
						}
						t.SetInt(n)
						mFieldValue.Set(t)
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
						t := reflect.New(mFieldType.Type.Elem())
						n, err := strconv.ParseUint(val, 10, 64)
						if err != nil || mFieldValue.OverflowUint(n) {
							return fmt.Errorf("failed to parse uint %s: %s", key, val)
						}
						t.SetUint(n)
						mFieldValue.Set(t)
					case reflect.Float32, reflect.Float64:
						t := reflect.New(mFieldType.Type.Elem())
						n, err := strconv.ParseFloat(val, mFieldValue.Type().Bits())
						if err != nil || mFieldValue.OverflowFloat(n) {
							return fmt.Errorf("failed to parse float %s: %s", key, val)
						}
						t.Elem().SetFloat(n)
						mFieldValue.Set(t)
					}
				}
			}
		}
	}
	return nil
}

// UnmarshalGetQuery parses the parameters of a GET request, and applies them as
// Predicates for a rest.Query
func UnmarshalGetQuery(r *http.Request, m Model, q Query) error {
	var operators = make(map[string]string)
	newURL := *r.URL
	newQuery := newURL.Query()
	for key, val := range r.URL.Query() {
		for suffix, operator := range queryMap {
			if strings.HasSuffix(key, suffix) {
				shortKey := key[:len(key)-len(suffix)]
				for _, s := range val {
					newQuery.Add(key[:len(key)-len(suffix)], s)
				}
				operators[shortKey] = operator
				newQuery.Del(key)
				break
			}
		}
	}
	newURL.RawQuery = newQuery.Encode()
	r.URL = &newURL
	mdl := m.New("")
	mdls, err := UnmarshalMultiQuery(r, mdl)
	if err != nil {
		return err
	}
	for _, mdl := range mdls {
		qModelValue := reflect.ValueOf(mdl).Elem()
		setFieldOperators(q, qModelValue, operators)
	}
	var qm queryModifiers
	err = UnmarshalQuery(r, &qm)
	if err != nil {
		return err
	}
	q.Skip(qm.Skip)
	if qm.Limit != 0 {
		q.Limit(qm.Limit)
	}
	if qm.Sort != "" {
		q.Sort(qm.Sort)
	}
	if qm.Has != "" {
		q.Has(qm.Has)
	}
	return nil
}

// queryModifiers are non Model-specific parameters for api queries
type queryModifiers struct {
	Skip  int    `json:"skip"`
	Limit int    `json:"limit"`
	Sort  string `json:"sort"`
	Has   string `json:"has"`
}

func setFieldOperators(q Query, mValue reflect.Value, operators map[string]string) {
	mType := mValue.Type()
	for i := 0; i < mValue.NumField(); i++ {
		tag := strings.Split(mType.Field(i).Tag.Get("json"), ",")[0]
		if tag == "" && mValue.Field(i).Kind() == reflect.Struct {
			setFieldOperators(q, mValue.Field(i), operators)
		}
		if _, ok := operators[tag]; ok {
			switch operators[tag] {
			case OperatorGreaterThan:
				q.GreaterThan(tag, mValue.Field(i).Interface())
			case OperatorLessThan:
				q.LessThan(tag, mValue.Field(i).Interface())
			case OperatorNotEqual:
				q.NotEqual(tag, mValue.Field(i).Interface())
			}
		} else {
			if !reflect.DeepEqual(mValue.Field(i).Interface(), reflect.Zero(mType.Field(i).Type).Interface()) && tag != "" {
				q.Equal(tag, mValue.Field(i).Interface())
			}
		}
	}
}

func clearContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		context.Clear(r)
	})
}

// TruePtr is a helper to return a pointer to a true boolean for queriable boolean fields
func TruePtr() *bool {
	var b = true
	return &b
}

// FalsePtr is a helper to return a pointer to a false boolean for queriable boolean fields
func FalsePtr() *bool {
	var b = false
	return &b
}
