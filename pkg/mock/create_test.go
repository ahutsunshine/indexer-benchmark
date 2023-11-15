package mock

import (
	"reflect"
	"testing"
)

func TestSplitGroups(t *testing.T) {
	threads := 10
	objNum := 1
	actual := splitGroups(threads, objNum)
	expected := []int{1}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("not equal, expected: %v,actual: %v", expected, actual)
	}

	objNum = 10
	actual = splitGroups(threads, objNum)
	expected = []int{10}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("not equal, expected: %v,actual: %v", expected, actual)
	}

	objNum = 25
	actual = splitGroups(threads, objNum)
	expected = []int{10, 10, 5}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("not equal, expected: %v,actual: %v", expected, actual)
	}
}
