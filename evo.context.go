package evo

import (
	"encoding/json"
	"fmt"
	"github.com/getevo/evo-min/lib/log"
	"github.com/gofiber/fiber/v2"
	"net/url"
	"reflect"
	"strings"
	"time"

	e "github.com/getevo/evo-min/errors"
)

type Request struct {
	Variables     fiber.Map
	Context       *fiber.Ctx
	Additional    interface{}
	Response      Response
	CacheKey      string
	CacheDuration time.Duration
	Debug         bool
	flashes       []flash
	BeforeWrite   func(request *Request, body []byte) []byte
}
type flash struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Error   e.Errors    `json:"errors"`
	Data    interface{} `json:"data"`
	Code    int         `json:"code"`
}

type URL struct {
	Query  url.Values
	Host   string
	Scheme string
	Path   string
	Raw    string
}

func (response Response) HasError() bool {
	return response.Error.Exist()
}

func (r *Request) URL() *URL {
	u := URL{}
	base := strings.Split(r.BaseURL(), "://")
	u.Scheme = base[0]
	u.Host = r.Hostname()
	u.Raw = r.OriginalURL()
	parts := strings.Split(u.Raw, "?")
	if len(parts) == 1 {
		u.Query = url.Values{}
		u.Path = u.Raw
	} else {
		u.Path = parts[0]
		u.Query, _ = url.ParseQuery(strings.Join(parts[1:], "?"))
	}
	return &u
}
func (u *URL) Set(key string, value interface{}) *URL {
	u.Query.Set(key, fmt.Sprint(value))
	return u
}
func (u *URL) String() string {
	return u.Path + "?" + u.Query.Encode()
}

func Upgrade(ctx *fiber.Ctx) *Request {
	if request := ctx.Locals("evo.request"); request != nil {
		return request.(*Request)
	}
	r := Request{}
	r.Variables = fiber.Map{}
	r.Context = ctx
	r.Response = Response{}
	r.Response.Error = e.Errors{}
	ctx.Locals("evo.request", &r)
	return &r
}

func (r *Request) Flash(params ...string) {
	if len(params) == 0 {
		return
	}
	if len(r.flashes) == 0 {
		cookie := r.Cookies("flash")
		if cookie != "" {
			json.Unmarshal([]byte(cookie), &r.flashes)
		}
	}
	if len(params) == 1 {
		r.flashes = append(r.flashes, flash{"info", params[0]})

	} else {
		r.flashes = append(r.flashes, flash{params[0], params[1]})
	}
	r.SetCookie("flash", r.flashes)
}

func (r *Request) Cached(duration time.Duration, key ...string) bool {
	r.CacheKey = ""
	r.CacheDuration = duration
	for _, item := range key {
		r.CacheKey += item
	}
	if v, ok := Cache.Get(r.CacheKey); ok {
		if resp, ok := v.(cached); ok {
			r.Context.Context().Response.Header = resp.header
			r.Context.Context().SetStatusCode(resp.code)
			r.Context.Context().Response.SetBody(resp.content)

			return true
		}
	}
	return false
}

func (r *Request) WriteResponse(resp ...interface{}) {
	if len(resp) == 0 {
		r._writeResponse(r.Response)
		return
	}
	var message = false
	for _, item := range resp {
		ref := reflect.ValueOf(item)

		switch ref.Kind() {
		case reflect.Struct:
			if v, ok := item.(Response); ok {
				r._writeResponse(v)
				return
			} else if v, ok := item.(e.Error); ok {
				r.Response.Error.Push(&v)
				r._writeResponse(r.Response)
			} else {
				r.Response.Success = true
				r.Response.Data = item
			}

		case reflect.Ptr:
			obj := ref.Elem().Interface()
			if v, ok := obj.(Response); ok {
				r._writeResponse(v)
				return
			} else if v, ok := obj.(e.Error); ok {
				r.Response.Error.Push(&v)
			} else if v, ok := item.(error); ok {

				r.Response.Error.Push(e.Context(v.Error()))
			} else {
				r.Response.Data = obj
			}

			break
		case reflect.Bool:
			r.Response.Success = item.(bool)
			break
		case reflect.Int32, reflect.Int16, reflect.Int64:
			if r.Response.Code == 0 || len(r.Response.Error) > 0 {
				r.Response.Code = item.(int)
			} else {
				r.Response.Data = item.(int)
			}
			break
		case reflect.String:
			if !message {
				r.Response.Message = item.(string)
				message = true
			} else {
				r.Response.Data = item
			}
			break
		default:
			r.Response.Data = item
			r.Response.Success = true
		}

	}
	r._writeResponse(r.Response)

}

func (r *Request) _writeResponse(resp Response) {
	if resp.HasError() {
		r.Response.Success = false
	} else {
		r.Response.Success = true
	}
	r.JSON(r.Response)
}

func (r *Request) SetError(err interface{}) {
	if v, ok := err.(error); ok {
		r.Response.Error.Push(e.Context(v.Error()))
		return
	}
	if v, ok := err.(e.Error); ok {
		r.Response.Error.Push(&v)
		return
	}
	if v, ok := err.(*e.Error); ok {
		r.Response.Error.Push(v)
		return
	}
	log.Error("invalid error provided %+v", err)

}

func (r *Request) Throw(e *e.Error) {
	r.Response.Error.Push(e)
	r.WriteResponse()
}

func (r *Request) HasError() bool {
	return r.Response.Error.Exist()
}

func (r *Request) Var(key string, value interface{}) {
	r.Variables[key] = value
}

func (r Request) GetFlashes() []flash {
	var resp []flash
	flash := r.Cookies("flash")
	if flash != "" {
		json.Unmarshal([]byte(flash), &resp)
	}

	return resp
}
