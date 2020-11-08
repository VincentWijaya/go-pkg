package curl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type IHttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type IHttpRequestor interface {
	NewHttpRequest(method string, uri string) IHttpRequest
}

type IHttpRequest interface {
	SetHeader(key, value string)
	SetBody(body []byte)
	SetParam(params url.Values)
	AddParam(key, value string)
	AddFile(key string, fileName string, value io.ReadWriteCloser)
	Do(ctx context.Context, timeout int) (IHttpResponse, error)
	String() string
}

type IHttpResponse interface {
	Is(statusCode int) bool
	IsSuccess() bool
	GetStatusCode() int
	GetBody() []byte
	String() string
}

type httpFile struct {
	fileName    string
	fileContent io.ReadWriteCloser
}

type HttpRequestor struct {
	client IHttpClient
}

type HttpRequest struct {
	client    IHttpClient
	method    string
	url       string
	headers   map[string]string
	params    url.Values
	files     map[string]httpFile
	body      []byte
	multipart bool
}

type HttpResponse struct {
	response *http.Response
	body     []byte
}

// NewHTTPClient new HTTP Client
func NewHTTPClient() IHttpClient {
	return &http.Client{}
}

func NewHttpRequestor(client IHttpClient) IHttpRequestor {
	return &HttpRequestor{client: client}
}

func (rq *HttpRequestor) NewHttpRequest(method string, uri string) IHttpRequest {
	return &HttpRequest{
		client:  rq.client,
		method:  strings.ToUpper(method),
		url:     uri,
		headers: map[string]string{},
		params:  url.Values{},
		files:   map[string]httpFile{},
	}
}

func isValidMethod(method string) bool {
	validMethod := []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, v := range validMethod {
		if v == method {
			return true
		}
	}
	return false
}

func (rq *HttpRequest) setQueryParams(u *url.URL) (*http.Request, error) {
	if len(rq.params) != 0 {
		query := u.Query()
		for k, _ := range rq.params {
			query.Add(k, rq.params.Get(k))
		}
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequest(rq.method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (rq *HttpRequest) setImageParams() (io.Reader, error) {
	var form bytes.Buffer
	var err error
	writer := multipart.NewWriter(&form)

	for key, value := range rq.files {
		defer value.fileContent.Close()

		var fw io.Writer
		if fw, err = writer.CreateFormFile(key, value.fileName); err != nil {
			return nil, fmt.Errorf("Failed to Create Form File. Error: %s", err)
		}
		if _, err = io.Copy(fw, value.fileContent); err != nil {
			return nil, fmt.Errorf("Failed to copy file to writer")
		}
	}

	for key, _ := range rq.params {
		var fw io.Writer
		if fw, err = writer.CreateFormField(key); err != nil {
			return nil, fmt.Errorf("Failed to Create Form Field. Error: %s", err)
		}
		if _, err = io.Copy(fw, strings.NewReader(rq.params.Get(key))); err != nil {
			return nil, fmt.Errorf("Failed to copy field to writer")
		}
	}

	rq.SetHeader("Content-Type", writer.FormDataContentType())
	writer.Close()

	return &form, nil
}

func (rq *HttpRequest) setBodyParams(u *url.URL) (*http.Request, error) {
	var err error
	var form io.Reader
	if len(rq.files) != 0 || rq.multipart {
		form, err = rq.setImageParams()
		if err != nil {
			return nil, err
		}
	} else if len(rq.body) != 0 {
		form = strings.NewReader(string(rq.body))
	} else {
		form = strings.NewReader(rq.params.Encode())
		rq.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	}

	req, err := http.NewRequest(rq.method, u.String(), form)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (rq *HttpRequest) SetHeader(key, value string) {
	rq.headers[key] = value

	key = strings.ToLower(key)
	if key == "content-type" && strings.HasPrefix(value, "multipart") {
		rq.multipart = true
	} else if key == "content-type" {
		rq.multipart = false
	}
}

func (rq *HttpRequest) SetBody(body []byte) {
	rq.body = body
}

func (rq *HttpRequest) SetParam(params url.Values) {
	rq.params = params
}

func (rq *HttpRequest) AddParam(key, value string) {
	rq.params.Add(key, value)
}

func (rq *HttpRequest) AddFile(key string, fileName string, value io.ReadWriteCloser) {
	rq.files[key] = httpFile{fileName: fileName, fileContent: value}
}

func (rq *HttpRequest) Do(ctx context.Context, timeout int) (IHttpResponse, error) {
	u, err := url.Parse(rq.url)
	if err != nil {
		return nil, err
	}

	if !isValidMethod(rq.method) {
		return nil, err
	}

	var request *http.Request
	if rq.method == http.MethodGet {
		request, err = rq.setQueryParams(u)
	} else {
		request, err = rq.setBodyParams(u)
	}
	if err != nil {
		return nil, err
	}

	if timeout > 0 {
		ctx, _ = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	}
	request = request.WithContext(ctx)

	for key, value := range rq.headers {
		request.Header.Set(key, value)
	}

	response, err := rq.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return &HttpResponse{response: response, body: contents}, nil
}

func (rq *HttpRequest) String() string {
	body := rq.params.Encode()
	if len(rq.body) > 0 {
		body = string(rq.body)
	} else if len(rq.files) > 0 {
		for key, value := range rq.files {
			body = fmt.Sprintf("%s&%s=%s", body, key, value.fileName)
		}
	}

	unescapedBody, err := url.QueryUnescape(body)
	if err == nil {
		body = unescapedBody
	}
	return fmt.Sprintf("Request %s to %s with header: %+v and body: %s", rq.method, rq.url, rq.headers, body)
}

func (rs *HttpResponse) Is(statusCode int) bool {
	if rs.response.StatusCode == statusCode {
		return true
	}
	return false
}

func (rs *HttpResponse) IsSuccess() bool {
	if rs.response.StatusCode == 200 || rs.response.StatusCode == 201 || rs.response.StatusCode == 204 {
		return true
	}

	return false
}

func (rs *HttpResponse) GetStatusCode() int {
	return rs.response.StatusCode
}

func (rs *HttpResponse) GetBody() []byte {
	return rs.body
}

func (rs *HttpResponse) String() string {
	return fmt.Sprintf("Response from %s with body: %s", rs.response.Request.URL.String(), strings.Replace(string(rs.body), "\n", "", -1))
}
