// Code generated by protoc-gen-gohttp. DO NOT EDIT.
// source: api.proto

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StatisticsHTTPConverter has a function to convert StatisticsServer interface to http.HandlerFunc.
type StatisticsHTTPConverter struct {
	srv StatisticsServer
}

// NewStatisticsHTTPConverter returns StatisticsHTTPConverter.
func NewStatisticsHTTPConverter(srv StatisticsServer) *StatisticsHTTPConverter {
	return &StatisticsHTTPConverter{
		srv: srv,
	}
}

// GetDashboard returns StatisticsServer interface's GetDashboard converted to http.HandlerFunc.
func (h *StatisticsHTTPConverter) GetDashboard(cb func(ctx context.Context, w http.ResponseWriter, r *http.Request, arg, ret proto.Message, err error), interceptors ...grpc.UnaryServerInterceptor) http.HandlerFunc {
	if cb == nil {
		cb = func(ctx context.Context, w http.ResponseWriter, r *http.Request, arg, ret proto.Message, err error) {
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				p := status.New(codes.Unknown, err.Error()).Proto()
				switch r.Header.Get("Content-Type") {
				case "application/protobuf", "application/x-protobuf":
					buf, err := proto.Marshal(p)
					if err != nil {
						return
					}
					if _, err := io.Copy(w, bytes.NewBuffer(buf)); err != nil {
						return
					}
				case "application/json":
					if err := json.NewEncoder(w).Encode(p); err != nil {
						return
					}
				default:
				}
			}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		arg := &GetDashboardRequest{}
		contentType := r.Header.Get("Content-Type")
		if r.Method != http.MethodGet {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				cb(ctx, w, r, nil, nil, err)
				return
			}

			switch contentType {
			case "application/protobuf", "application/x-protobuf":
				if err := proto.Unmarshal(body, arg); err != nil {
					cb(ctx, w, r, nil, nil, err)
					return
				}
			case "application/json":
				if err := jsonpb.Unmarshal(bytes.NewBuffer(body), arg); err != nil {
					cb(ctx, w, r, nil, nil, err)
					return
				}
			default:
				w.WriteHeader(http.StatusUnsupportedMediaType)
				_, err := fmt.Fprintf(w, "Unsupported Content-Type: %s", contentType)
				cb(ctx, w, r, nil, nil, err)
				return
			}
		}

		n := len(interceptors)
		chained := func(ctx context.Context, arg interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			chainer := func(currentInter grpc.UnaryServerInterceptor, currentHandler grpc.UnaryHandler) grpc.UnaryHandler {
				return func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
					return currentInter(currentCtx, currentReq, info, currentHandler)
				}
			}

			chainedHandler := handler
			for i := n - 1; i >= 0; i-- {
				chainedHandler = chainer(interceptors[i], chainedHandler)
			}
			return chainedHandler(ctx, arg)
		}

		info := &grpc.UnaryServerInfo{
			Server:     h.srv,
			FullMethod: "/api.Statistics/GetDashboard",
		}

		handler := func(c context.Context, req interface{}) (interface{}, error) {
			return h.srv.GetDashboard(c, req.(*GetDashboardRequest))
		}

		iret, err := chained(ctx, arg, info, handler)
		if err != nil {
			cb(ctx, w, r, arg, nil, err)
			return
		}

		ret, ok := iret.(*GetDashboardResponse)
		if !ok {
			cb(ctx, w, r, arg, nil, fmt.Errorf("/api.Statistics/GetDashboard: interceptors have not return GetDashboardResponse"))
			return
		}

		accepts := strings.Split(r.Header.Get("Accept"), ",")
		accept := accepts[0]
		if accept == "*/*" || accept == "" {
			if contentType != "" {
				accept = contentType
			} else {
				accept = "application/json"
			}
		}

		w.Header().Set("Content-Type", accept)

		switch accept {
		case "application/protobuf", "application/x-protobuf":
			buf, err := proto.Marshal(ret)
			if err != nil {
				cb(ctx, w, r, arg, ret, err)
				return
			}
			if _, err := io.Copy(w, bytes.NewBuffer(buf)); err != nil {
				cb(ctx, w, r, arg, ret, err)
				return
			}
		case "application/json":
			m := jsonpb.Marshaler{
				EnumsAsInts:  true,
				EmitDefaults: true,
			}
			if err := m.Marshal(w, ret); err != nil {
				cb(ctx, w, r, arg, ret, err)
				return
			}
		default:
			w.WriteHeader(http.StatusUnsupportedMediaType)
			_, err := fmt.Fprintf(w, "Unsupported Accept: %s", accept)
			cb(ctx, w, r, arg, ret, err)
			return
		}
		cb(ctx, w, r, arg, ret, nil)
	})
}

// GetDashboardWithName returns Service name, Method name and StatisticsServer interface's GetDashboard converted to http.HandlerFunc.
func (h *StatisticsHTTPConverter) GetDashboardWithName(cb func(ctx context.Context, w http.ResponseWriter, r *http.Request, arg, ret proto.Message, err error), interceptors ...grpc.UnaryServerInterceptor) (string, string, http.HandlerFunc) {
	return "Statistics", "GetDashboard", h.GetDashboard(cb, interceptors...)
}

// ListWorks returns StatisticsServer interface's ListWorks converted to http.HandlerFunc.
func (h *StatisticsHTTPConverter) ListWorks(cb func(ctx context.Context, w http.ResponseWriter, r *http.Request, arg, ret proto.Message, err error), interceptors ...grpc.UnaryServerInterceptor) http.HandlerFunc {
	if cb == nil {
		cb = func(ctx context.Context, w http.ResponseWriter, r *http.Request, arg, ret proto.Message, err error) {
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				p := status.New(codes.Unknown, err.Error()).Proto()
				switch r.Header.Get("Content-Type") {
				case "application/protobuf", "application/x-protobuf":
					buf, err := proto.Marshal(p)
					if err != nil {
						return
					}
					if _, err := io.Copy(w, bytes.NewBuffer(buf)); err != nil {
						return
					}
				case "application/json":
					if err := json.NewEncoder(w).Encode(p); err != nil {
						return
					}
				default:
				}
			}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		arg := &ListWorksRequest{}
		contentType := r.Header.Get("Content-Type")
		if r.Method != http.MethodGet {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				cb(ctx, w, r, nil, nil, err)
				return
			}

			switch contentType {
			case "application/protobuf", "application/x-protobuf":
				if err := proto.Unmarshal(body, arg); err != nil {
					cb(ctx, w, r, nil, nil, err)
					return
				}
			case "application/json":
				if err := jsonpb.Unmarshal(bytes.NewBuffer(body), arg); err != nil {
					cb(ctx, w, r, nil, nil, err)
					return
				}
			default:
				w.WriteHeader(http.StatusUnsupportedMediaType)
				_, err := fmt.Fprintf(w, "Unsupported Content-Type: %s", contentType)
				cb(ctx, w, r, nil, nil, err)
				return
			}
		}

		n := len(interceptors)
		chained := func(ctx context.Context, arg interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			chainer := func(currentInter grpc.UnaryServerInterceptor, currentHandler grpc.UnaryHandler) grpc.UnaryHandler {
				return func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
					return currentInter(currentCtx, currentReq, info, currentHandler)
				}
			}

			chainedHandler := handler
			for i := n - 1; i >= 0; i-- {
				chainedHandler = chainer(interceptors[i], chainedHandler)
			}
			return chainedHandler(ctx, arg)
		}

		info := &grpc.UnaryServerInfo{
			Server:     h.srv,
			FullMethod: "/api.Statistics/ListWorks",
		}

		handler := func(c context.Context, req interface{}) (interface{}, error) {
			return h.srv.ListWorks(c, req.(*ListWorksRequest))
		}

		iret, err := chained(ctx, arg, info, handler)
		if err != nil {
			cb(ctx, w, r, arg, nil, err)
			return
		}

		ret, ok := iret.(*ListWorksResponse)
		if !ok {
			cb(ctx, w, r, arg, nil, fmt.Errorf("/api.Statistics/ListWorks: interceptors have not return ListWorksResponse"))
			return
		}

		accepts := strings.Split(r.Header.Get("Accept"), ",")
		accept := accepts[0]
		if accept == "*/*" || accept == "" {
			if contentType != "" {
				accept = contentType
			} else {
				accept = "application/json"
			}
		}

		w.Header().Set("Content-Type", accept)

		switch accept {
		case "application/protobuf", "application/x-protobuf":
			buf, err := proto.Marshal(ret)
			if err != nil {
				cb(ctx, w, r, arg, ret, err)
				return
			}
			if _, err := io.Copy(w, bytes.NewBuffer(buf)); err != nil {
				cb(ctx, w, r, arg, ret, err)
				return
			}
		case "application/json":
			m := jsonpb.Marshaler{
				EnumsAsInts:  true,
				EmitDefaults: true,
			}
			if err := m.Marshal(w, ret); err != nil {
				cb(ctx, w, r, arg, ret, err)
				return
			}
		default:
			w.WriteHeader(http.StatusUnsupportedMediaType)
			_, err := fmt.Fprintf(w, "Unsupported Accept: %s", accept)
			cb(ctx, w, r, arg, ret, err)
			return
		}
		cb(ctx, w, r, arg, ret, nil)
	})
}

// ListWorksWithName returns Service name, Method name and StatisticsServer interface's ListWorks converted to http.HandlerFunc.
func (h *StatisticsHTTPConverter) ListWorksWithName(cb func(ctx context.Context, w http.ResponseWriter, r *http.Request, arg, ret proto.Message, err error), interceptors ...grpc.UnaryServerInterceptor) (string, string, http.HandlerFunc) {
	return "Statistics", "ListWorks", h.ListWorks(cb, interceptors...)
}
