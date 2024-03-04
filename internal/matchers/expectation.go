// Code created by gotmpl. DO NOT MODIFY.
// source: internal/shared/matchers/expectation.go.tmpl

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package matchers // import "go.opentelemetry.io/otel/internal/matchers"

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

var stackTracePruneRE = regexp.MustCompile(`runtime\/debug|testing|internal\/matchers`)

type Expectation struct {
	t      *testing.T
	actual interface{}
}

func (e *Expectation) ToEqual(expected interface{}) {
	e.verifyExpectedNotNil(expected)

	if !reflect.DeepEqual(e.actual, expected) {
		e.fail(fmt.Sprintf("Expected\n\t%v\nto equal\n\t%v", e.actual, expected))
	}
}

func (e *Expectation) NotToEqual(expected interface{}) {
	e.verifyExpectedNotNil(expected)

	if reflect.DeepEqual(e.actual, expected) {
		e.fail(fmt.Sprintf("Expected\n\t%v\nnot to equal\n\t%v", e.actual, expected))
	}
}

func (e *Expectation) ToBeNil() {
	if e.actual != nil {
		e.fail(fmt.Sprintf("Expected\n\t%v\nto be nil", e.actual))
	}
}

func (e *Expectation) NotToBeNil() {
	if e.actual == nil {
		e.fail(fmt.Sprintf("Expected\n\t%v\nnot to be nil", e.actual))
	}
}

func (e *Expectation) ToBeTrue() {
	switch a := e.actual.(type) {
	case bool:
		if !a {
			e.fail(fmt.Sprintf("Expected\n\t%v\nto be true", e.actual))
		}
	default:
		e.fail(fmt.Sprintf("Cannot check if non-bool value\n\t%v\nis truthy", a))
	}
}

func (e *Expectation) ToBeFalse() {
	switch a := e.actual.(type) {
	case bool:
		if a {
			e.fail(fmt.Sprintf("Expected\n\t%v\nto be false", e.actual))
		}
	default:
		e.fail(fmt.Sprintf("Cannot check if non-bool value\n\t%v\nis truthy", a))
	}
}

func (e *Expectation) NotToPanic() {
	switch a := e.actual.(type) {
	case func():
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					e.fail(fmt.Sprintf("Expected panic\n\t%v\nto have not been raised", recovered))
				}
			}()

			a()
		}()
	default:
		e.fail(fmt.Sprintf("Cannot check if non-func value\n\t%v\nis truthy", a))
	}
}

func (e *Expectation) ToSucceed() {
	switch actual := e.actual.(type) {
	case error:
		if actual != nil {
			e.fail(fmt.Sprintf("Expected error\n\t%v\nto have succeeded", actual))
		}
	default:
		e.fail(fmt.Sprintf("Cannot check if non-error value\n\t%v\nsucceeded", actual))
	}
}

func (e *Expectation) ToMatchError(expected interface{}) {
	e.verifyExpectedNotNil(expected)

	actual, ok := e.actual.(error)
	if !ok {
		e.fail(fmt.Sprintf("Cannot check if non-error value\n\t%v\nmatches error", e.actual))
	}

	switch expected := expected.(type) {
	case error:
		if !reflect.DeepEqual(actual, expected) {
			e.fail(fmt.Sprintf("Expected\n\t%v\nto match error\n\t%v", actual, expected))
		}
	case string:
		if actual.Error() != expected {
			e.fail(fmt.Sprintf("Expected\n\t%v\nto match error\n\t%v", actual, expected))
		}
	default:
		e.fail(fmt.Sprintf("Cannot match\n\t%v\nagainst non-error\n\t%v", actual, expected))
	}
}

func (e *Expectation) ToContain(expected interface{}) {
	actualValue := reflect.ValueOf(e.actual)
	actualKind := actualValue.Kind()

	switch actualKind {
	case reflect.Array, reflect.Slice:
	default:
		e.fail(fmt.Sprintf("Expected\n\t%v\nto be an array", e.actual))
		return
	}

	expectedValue := reflect.ValueOf(expected)
	expectedKind := expectedValue.Kind()

	switch expectedKind {
	case reflect.Array, reflect.Slice:
	default:
		expectedValue = reflect.ValueOf([]interface{}{expected})
	}

	for i := 0; i < expectedValue.Len(); i++ {
		var contained bool
		expectedElem := expectedValue.Index(i).Interface()

		for j := 0; j < actualValue.Len(); j++ {
			if reflect.DeepEqual(actualValue.Index(j).Interface(), expectedElem) {
				contained = true
				break
			}
		}

		if !contained {
			e.fail(fmt.Sprintf("Expected\n\t%v\nto contain\n\t%v", e.actual, expectedElem))
			return
		}
	}
}

func (e *Expectation) NotToContain(expected interface{}) {
	actualValue := reflect.ValueOf(e.actual)
	actualKind := actualValue.Kind()

	switch actualKind {
	case reflect.Array, reflect.Slice:
	default:
		e.fail(fmt.Sprintf("Expected\n\t%v\nto be an array", e.actual))
		return
	}

	expectedValue := reflect.ValueOf(expected)
	expectedKind := expectedValue.Kind()

	switch expectedKind {
	case reflect.Array, reflect.Slice:
	default:
		expectedValue = reflect.ValueOf([]interface{}{expected})
	}

	for i := 0; i < expectedValue.Len(); i++ {
		expectedElem := expectedValue.Index(i).Interface()

		for j := 0; j < actualValue.Len(); j++ {
			if reflect.DeepEqual(actualValue.Index(j).Interface(), expectedElem) {
				e.fail(fmt.Sprintf("Expected\n\t%v\nnot to contain\n\t%v", e.actual, expectedElem))
				return
			}
		}
	}
}

func (e *Expectation) ToMatchInAnyOrder(expected interface{}) {
	expectedValue := reflect.ValueOf(expected)
	expectedKind := expectedValue.Kind()

	switch expectedKind {
	case reflect.Array, reflect.Slice:
	default:
		e.fail(fmt.Sprintf("Expected\n\t%v\nto be an array", expected))
		return
	}

	actualValue := reflect.ValueOf(e.actual)
	actualKind := actualValue.Kind()

	if actualKind != expectedKind {
		e.fail(fmt.Sprintf("Expected\n\t%v\nto be the same type as\n\t%v", e.actual, expected))
		return
	}

	if actualValue.Len() != expectedValue.Len() {
		e.fail(fmt.Sprintf("Expected\n\t%v\nto have the same length as\n\t%v", e.actual, expected))
		return
	}

	var unmatched []interface{}

	for i := 0; i < expectedValue.Len(); i++ {
		unmatched = append(unmatched, expectedValue.Index(i).Interface())
	}

	for i := 0; i < actualValue.Len(); i++ {
		var found bool

		for j, elem := range unmatched {
			if reflect.DeepEqual(actualValue.Index(i).Interface(), elem) {
				found = true
				unmatched = append(unmatched[:j], unmatched[j+1:]...)

				break
			}
		}

		if !found {
			e.fail(fmt.Sprintf("Expected\n\t%v\nto contain the same elements as\n\t%v", e.actual, expected))
		}
	}
}

func (e *Expectation) ToBeTemporally(matcher TemporalMatcher, compareTo interface{}) {
	if actual, ok := e.actual.(time.Time); ok {
		ct, ok := compareTo.(time.Time)
		if !ok {
			e.fail(fmt.Sprintf("Cannot compare to non-temporal value\n\t%v", compareTo))
			return
		}

		switch matcher {
		case Before:
			if !actual.Before(ct) {
				e.fail(fmt.Sprintf("Expected\n\t%v\nto be temporally before\n\t%v", e.actual, compareTo))
			}
		case BeforeOrSameTime:
			if actual.After(ct) {
				e.fail(fmt.Sprintf("Expected\n\t%v\nto be temporally before or at the same time as\n\t%v", e.actual, compareTo))
			}
		case After:
			if !actual.After(ct) {
				e.fail(fmt.Sprintf("Expected\n\t%v\nto be temporally after\n\t%v", e.actual, compareTo))
			}
		case AfterOrSameTime:
			if actual.Before(ct) {
				e.fail(fmt.Sprintf("Expected\n\t%v\nto be temporally after or at the same time as\n\t%v", e.actual, compareTo))
			}
		default:
			e.fail("Cannot compare times with unexpected temporal matcher")
		}

		return
	}

	e.fail(fmt.Sprintf("Cannot compare non-temporal value\n\t%v", e.actual))
}

func (e *Expectation) verifyExpectedNotNil(expected interface{}) {
	if expected == nil {
		e.fail("Refusing to compare with <nil>. Use `ToBeNil` or `NotToBeNil` instead.")
	}
}

func (e *Expectation) fail(msg string) {
	// Prune the stack trace so that it's easier to see relevant lines
	stack := strings.Split(string(debug.Stack()), "\n")
	var prunedStack []string

	for _, line := range stack {
		if !stackTracePruneRE.MatchString(line) {
			prunedStack = append(prunedStack, line)
		}
	}

	e.t.Fatalf("\n%s\n%s\n", strings.Join(prunedStack, "\n"), msg)
}
