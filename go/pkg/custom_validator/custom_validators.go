package custom_validator

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type GameLibraryTagCategoryType string

const (
	PLATFORM               GameLibraryTagCategoryType = "PLATFORM"
	BRANDING_POSSIBILITIES GameLibraryTagCategoryType = "BRANDING_POSSIBILITIES"
	GAME_CATEGORIES        GameLibraryTagCategoryType = "GAME_CATEGORIES"
	TAGS                   GameLibraryTagCategoryType = "TAGS"
)

type GameLibraryTagDto struct {
	GameLibraryTagId string                     `validate:"required,string"`
	TagName          string                     `validate:"required,string"`
	TagId            *int                       `validate:"number,optional"`
	CategoryName     GameLibraryTagCategoryType `validate:"required,enum=PLATFORM|BRANDING_POSSIBILITIES|GAME_CATEGORIES|TAGS"`
}

func validateStruct(s any) error {
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)

	if t.Kind() != reflect.Struct {
		return errors.New("ValidateStruct only accepts structs")
	}

	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("validate")

		if tag == "" {
			continue
		}

		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			if err := applyRule(rule, fieldVal, fieldType); err != nil {
				return fmt.Errorf("field %s: %w", fieldType.Name, err)
			}
		}
	}
	return nil
}

func applyRule(rule string, val reflect.Value, f reflect.StructField) error {
	switch {
	case rule == "required":
		if isEmptyValue(val) {
			return errors.New("is required")
		}
	case rule == "string":
		if val.Kind() != reflect.String {
			return errors.New("must be a string")
		}
	case rule == "number":
		if val.Kind() != reflect.Int && val.Kind() != reflect.Ptr {
			return errors.New("must be a number")
		}
	case strings.HasPrefix(rule, "enum="):
		if val.Kind() != reflect.String {
			return errors.New("must be a string enum")
		}
		valids := strings.Split(strings.TrimPrefix(rule, "enum="), "|")
		if !contains(valids, val.String()) {
			return fmt.Errorf("must be one of %v", valids)
		}
	}
	return nil
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	}
	// zero value check
	return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
}

func contains(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}
