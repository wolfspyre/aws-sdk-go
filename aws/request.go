package aws

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/awslabs/aws-sdk-go/aws/awsutil"
)

type Request struct {
	*Service
	Handlers     Handlers
	Time         time.Time
	ExpireTime   time.Duration
	Operation    *Operation
	HTTPRequest  *http.Request
	HTTPResponse *http.Response
	Body         io.ReadSeeker
	Params       interface{}
	Error        error
	Data         interface{}
	RequestID    string
	RetryCount   uint

	built bool
}

type Operation struct {
	Name       string
	HTTPMethod string
	HTTPPath   string
	*Paginator
}

type Paginator struct {
	InputToken      string
	OutputToken     string
	LimitToken      string
	TruncationToken string
}

func NewRequest(service *Service, operation *Operation, params interface{}, data interface{}) *Request {
	method := operation.HTTPMethod
	if method == "" {
		method = "POST"
	}
	p := operation.HTTPPath
	if p == "" {
		p = "/"
	}

	httpReq, _ := http.NewRequest(method, "", nil)
	httpReq.URL, _ = url.Parse(service.Endpoint + p)

	r := &Request{
		Service:     service,
		Handlers:    service.Handlers.copy(),
		Time:        time.Now(),
		ExpireTime:  0,
		Operation:   operation,
		HTTPRequest: httpReq,
		Body:        nil,
		Params:      params,
		Error:       nil,
		Data:        data,
	}
	r.SetBufferBody([]byte{})

	return r
}

func (r *Request) ParamsFilled() bool {
	return r.Params != nil && reflect.ValueOf(r.Params).Elem().IsValid()
}

func (r *Request) DataFilled() bool {
	return r.Data != nil && reflect.ValueOf(r.Data).Elem().IsValid()
}

func (r *Request) SetBufferBody(buf []byte) {
	r.SetReaderBody(bytes.NewReader(buf))
}

func (r *Request) SetReaderBody(reader io.ReadSeeker) {
	r.HTTPRequest.Body = ioutil.NopCloser(reader)
	r.Body = reader
}

func (r *Request) Presign(expireTime time.Duration) (string, error) {
	r.ExpireTime = expireTime
	r.Sign()
	if r.Error != nil {
		return "", r.Error
	} else {
		return r.HTTPRequest.URL.String(), nil
	}
}

func (r *Request) Build() error {
	if !r.built {
		r.Error = nil
		r.Handlers.Validate.Run(r)
		if r.Error != nil {
			return r.Error
		}
		r.Handlers.Build.Run(r)
		r.built = true
	}

	return r.Error
}

func (r *Request) Sign() error {
	r.Build()
	if r.Error != nil {
		return r.Error
	}

	r.Handlers.Sign.Run(r)
	return r.Error
}

func (r *Request) Send() error {
	r.Sign()
	if r.Error != nil {
		return r.Error
	}

	for {
		r.Handlers.Send.Run(r)
		if r.Error != nil {
			return r.Error
		}

		r.Handlers.UnmarshalMeta.Run(r)
		r.Handlers.ValidateResponse.Run(r)
		if r.Error != nil {
			r.Handlers.Retry.Run(r)
			r.Handlers.AfterRetry.Run(r)
			if r.Error != nil {
				r.Handlers.UnmarshalError.Run(r)
				return r.Error
			}
			continue
		}

		r.Handlers.Unmarshal.Run(r)
		if r.Error != nil {
			return r.Error
		}

		return nil
	}
}

func (r *Request) HasNextPage() bool {
	return r.nextPageToken() != nil
}

func (r *Request) nextPageToken() interface{} {
	if r.Operation.Paginator == nil {
		return nil
	}
	v := awsutil.ValuesAtPath(r.Data, r.Operation.OutputToken)
	if v != nil && len(v) > 0 {
		return v[0]
	}
	return nil
}

func (r *Request) NextPage() *Request {
	token := r.nextPageToken()
	if token == nil {
		return nil
	}

	if r.Operation.TruncationToken != "" {
		tr := awsutil.ValuesAtPath(r.Data, r.Operation.TruncationToken)
		if tr == nil {
			return nil
		} else if len(tr) > 0 {
			switch v := tr[0].(type) {
			case bool:
				if v == false {
					return nil
				}
			}
		}
	}

	data := reflect.New(reflect.TypeOf(r.Data).Elem()).Interface()
	nr := NewRequest(r.Service, r.Operation, r.Params, data)

	awsutil.SetValueAtPath(nr.Params, nr.Operation.InputToken, token)
	return nr
}
