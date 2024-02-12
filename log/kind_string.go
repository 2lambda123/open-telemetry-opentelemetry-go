// Code generated by "stringer -type=Kind -trimprefix=Kind"; DO NOT EDIT.

package log

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[KindEmpty-0]
	_ = x[KindBool-1]
	_ = x[KindFloat64-2]
	_ = x[KindInt64-3]
	_ = x[KindString-4]
	_ = x[KindBytes-5]
	_ = x[KindList-6]
	_ = x[KindMap-7]
}

const _Kind_name = "EmptyBoolFloat64Int64StringBytesListMap"

var _Kind_index = [...]uint8{0, 5, 9, 16, 21, 27, 32, 36, 39}

func (i Kind) String() string {
	if i < 0 || i >= Kind(len(_Kind_index)-1) {
		return "Kind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Kind_name[_Kind_index[i]:_Kind_index[i+1]]
}
