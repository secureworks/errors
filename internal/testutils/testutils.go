package testutils

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// AssertMatch is a semantic test assertion for string regex matching.
// If the pattern is empty we match only with an empty string for
// simplicity.
func AssertMatch(t *testing.T, pattern, value string, pads ...string) {
	t.Helper()

	if pattern == "" {
		AssertEqual(t, pattern, value, pads...)
		return
	}

	pads = append(pads, "does not match")
	padding := strings.Join(pads, ": ")

	match, err := regexp.MatchString(pattern, value)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Errorf(
			"%s:\n string: %q\npattern: re%q",
			padding, value, pattern)
	}
}

// AssertErrorMessage is a semantic test assertion for error "message
// context" equality.
func AssertErrorMessage(t *testing.T, expected string, err error, pads ...string) {
	t.Helper()
	assertEquality(t, expected, err.Error(), true, pads...)
}

// AssertEqual is a semantic test assertion for object equality.
func AssertEqual(t *testing.T, expected interface{}, actual interface{}, pads ...string) {
	t.Helper()
	assertEquality(t, expected, actual, true, pads...)
}

// AssertNotEqual is a semantic test assertion for object equality.
func AssertNotEqual(t *testing.T, expected interface{}, actual interface{}, pads ...string) {
	t.Helper()
	assertEquality(t, expected, actual, false, pads...)
}

// AssertNil is a semantic test assertion for nility.
func AssertNil(t *testing.T, object interface{}, pads ...string) {
	t.Helper()
	assertNility(t, object, true, pads...)
}

// AssertNotNil is a semantic test assertion for nility.
func AssertNotNil(t *testing.T, object interface{}, pads ...string) {
	t.Helper()
	assertNility(t, object, false, pads...)
}

// AssertTrue is a semantic test assertion for object truthiness.
func AssertTrue(t *testing.T, object bool, pads ...string) {
	t.Helper()
	if !object {
		pads = append(pads, "is not true")
		padding := strings.Join(pads, ": ")
		t.Error(padding)
	}
}

// AssertFalse is a semantic test assertion for object truthiness.
func AssertFalse(t *testing.T, object bool, pads ...string) {
	t.Helper()
	if object {
		pads = append(pads, "is not false")
		padding := strings.Join(pads, ": ")
		t.Error(padding)
	}
}

// AssertLinesMatch breaks up multiple lines and matches each with a
// regex per.
func AssertLinesMatch(t *testing.T, arg interface{}, format string, expected interface{}) {
	t.Helper()

	got := fmt.Sprintf(format, arg)
	gotLines := strings.SplitN(got, "\n", -1)

	var wantLines []string
	switch want := expected.(type) {
	case string:
		wantLines = strings.SplitN(want, "\n", -1)
	case []string:
		wantLines = want
	default:
		t.Fatalf("bad expected value passed: only handles string and []string: %#v", expected)
	}

	if len(wantLines) != len(gotLines) {
		t.Errorf(
			"wantLines(%d) does not equal gotLines(%d):\n got: %q\nwant: %q",
			len(wantLines), len(gotLines), got, expected)
		return
	}

	for i, w := range wantLines {
		AssertMatch(t, w, gotLines[i], fmt.Sprintf("line %0d", i+1))
	}
}

// NOTE(PH): does not handle bytes well, update if we need to check
// them.
func assertEquality(t *testing.T, expected interface{}, actual interface{}, wantEqual bool, pads ...string) {
	t.Helper()

	if expected == nil && actual == nil && !wantEqual {
		pads = append(pads, "is equal")
		padding := strings.Join(pads, ": ")
		t.Errorf("%s:\nexpected: %s\n  actual: %s\n", padding, expected, actual)
	}

	isEqual := reflect.DeepEqual(expected, actual)
	if wantEqual && !isEqual {
		pads = append(pads, "not equal")
		padding := strings.Join(pads, ": ")
		t.Errorf("%s:\nexpected: %s\n  actual: %s\n", padding, expected, actual)
	}
	if !wantEqual && isEqual {
		pads = append(pads, "is equal")
		padding := strings.Join(pads, ": ")
		t.Errorf("%s:\nexpected: %s\n  actual: %s\n", padding, expected, actual)
	}
}

func assertNility(t *testing.T, object interface{}, wantNil bool, pads ...string) {
	t.Helper()

	isNil := object == nil
	if !isNil {
		value := reflect.ValueOf(object)
		isNilable := false
		switch value.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
			isNilable = true
		default:
		}
		if isNilable && value.IsNil() {
			isNil = true
		}
	}
	if wantNil && !isNil {
		pads = append(pads, "not nil")
		padding := strings.Join(pads, ": ")
		t.Errorf("%s: %s\n", padding, object)
	}
	if !wantNil && isNil {
		pads = append(pads, "is nil")
		padding := strings.Join(pads, ": ")
		t.Errorf("%s: %s\n", padding, object)
	}
}
