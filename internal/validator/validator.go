package validator

import (
	"fmt"
	"reflect"
)

func Validate(name string, deps ...any) error {
	for _, dep := range deps {
		if v := reflect.ValueOf(dep); v.IsNil() || v.IsZero() {
			return fmt.Errorf("missing required deps for component: %s", name)
		}
	}

	return nil
}
