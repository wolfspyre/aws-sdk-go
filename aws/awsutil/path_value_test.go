package awsutil_test

import (
	"testing"

	"github.com/awslabs/aws-sdk-go/aws/awsutil"
	"github.com/stretchr/testify/assert"
)

type Struct struct {
	A []Struct
	a []Struct
	B *Struct
	C string
}

var data = Struct{
	A: []Struct{Struct{C: "value1"}, Struct{C: "value2"}, Struct{C: "value3"}},
	a: []Struct{Struct{C: "value1"}, Struct{C: "value2"}, Struct{C: "value3"}},
	B: &Struct{B: &Struct{C: "terminal"}},
	C: "initial",
}

func TestValueAtPathSuccess(t *testing.T) {

	assert.Equal(t, "initial", awsutil.ValueAtPath(data, "C"))
	assert.Equal(t, "value1", awsutil.ValueAtPath(data, "A[0].C"))
	assert.Equal(t, "value2", awsutil.ValueAtPath(data, "A[1].C"))
	assert.Equal(t, "value3", awsutil.ValueAtPath(data, "A[2].C"))
	assert.Equal(t, "value3", awsutil.ValueAtPath(data, "A[-1].C"))
	assert.Equal(t, "terminal", awsutil.ValueAtPath(data, "B.B.C"))
}

func TestValueAtPathFailure(t *testing.T) {
	assert.Equal(t, nil, awsutil.ValueAtPath(data, "C.x"))
	assert.Equal(t, nil, awsutil.ValueAtPath(data, ".x"))
	assert.Equal(t, nil, awsutil.ValueAtPath(data, "X.Y.Z"))
	assert.Equal(t, nil, awsutil.ValueAtPath(data, "A[100].C"))
	assert.Equal(t, nil, awsutil.ValueAtPath(data, "A[3].C"))
	assert.Equal(t, nil, awsutil.ValueAtPath(data, "B.B.C.Z"))
	assert.Equal(t, nil, awsutil.ValueAtPath(data, "a[-1].C"))
	assert.Equal(t, nil, awsutil.ValueAtPath(nil, "A.B.C"))
}

func TestSetValueAtPathSuccess(t *testing.T) {
	var s Struct
	awsutil.SetValueAtPath(&s, "C", "test1")
	awsutil.SetValueAtPath(&s, "B.B.C", "test2")
	assert.Equal(t, "test1", s.C)
	assert.Equal(t, "test2", s.B.B.C)
}
