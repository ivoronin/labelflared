package main

import (
	"log"
	"os"
	"reflect"
	"strconv"
)

func setFieldDefaults[T any](obj *T) {
	fields := reflect.VisibleFields(reflect.TypeOf(obj).Elem())
	for _, field := range fields {
		defaultValue := field.Tag.Get("default")
		if defaultValue == "" {
			continue
		}
		setField(obj, field.Name, defaultValue)
	}
}

func setField[T any](obj *T, field string, value string) {
	fieldValue := reflect.ValueOf(obj).Elem().FieldByName(field)
	if !fieldValue.IsValid() {
		log.Panicf("returned value for field %s is not valid", field)
	}
	switch fieldValue.Interface().(type) {
	case int:
		intValue, _ := strconv.Atoi(value)
		fieldValue.SetInt(int64(intValue))
	case string:
		fieldValue.SetString(value)
	default:
		log.Panicf("unexpected type for field %s", field)
	}
}

func requireEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}

func defaultEnv(name string, defaultValue string) string {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue
	}
	return value
}
