package grpc

import (
	"context"
	"fmt"
	"reflect"

	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/server"
	"google.golang.org/grpc"
)

type rpcHandler struct {
	name        string
	handler     interface{}
	endpoints   []*registry.Endpoint
	serviceDesc grpc.ServiceDesc
	opts        server.HandlerOptions
}

func newRpcHandler(serviceName string, handler interface{}, opts ...server.HandlerOption) server.Handler {
	options := server.HandlerOptions{
		Metadata: make(map[string]map[string]string),
	}

	for _, o := range opts {
		o(&options)
	}

	// typ := reflect.TypeOf(handler)
	hdlr := reflect.ValueOf(handler)
	typ := reflect.Indirect(hdlr).Field(0).Type()
	name := reflect.Indirect(hdlr).Type().String()

	var endpoints []*registry.Endpoint

	var methods []grpc.MethodDesc
	var streams []grpc.StreamDesc

	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		if e := extractEndpoint(method); e != nil {
			for k, v := range options.Metadata[e.Name] {
				e.Metadata[k] = v
			}

			if e.Metadata["stream"] == "true" {
				streams = append(streams, grpc.StreamDesc{
					StreamName: e.Name,
					Handler: func(srv interface{}, stream grpc.ServerStream) error {

						inType := method.Type.In(1).Elem()
						in := reflect.New(inType)

						err := stream.RecvMsg(in.Interface())
						if err != nil {
							return err
						}

						// for i := 0; i < method.Type.NumIn(); i++ {
						// 	fmt.Println(method)
						// 	fmt.Println(method.Type.In(i))
						// }
						// inType := method.Type.In(0).Elem()
						// in := reflect.New(inType)

						// err := stream.RecvMsg(in.Interface())
						// if err != nil {
						// 	return err
						// }

						ret := reflect.ValueOf(srv).MethodByName(e.Name).Call([]reflect.Value{reflect.ValueOf(context.Background()), in, reflect.ValueOf(stream)})
						if err, ok := ret[0].Interface().(error); ok {
							return err
						}

						return nil
					},
					ServerStreams: true,
				})

			} else {
				methods = append(methods, grpc.MethodDesc{
					MethodName: e.Name,
					Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
						inType := method.Type.In(2).Elem()
						in := reflect.New(inType)
						if err := dec(in.Interface()); err != nil {
							return nil, err
						}

						outType := method.Type.In(3).Elem()
						out := reflect.New(outType)
						if interceptor == nil {
							ret := reflect.ValueOf(srv).MethodByName(e.Name).Call([]reflect.Value{reflect.ValueOf(ctx), in, out})

							if err, ok := ret[0].Interface().(error); ok {
								return nil, err
							}
							return out.Interface(), nil
						}

						info := &grpc.UnaryServerInfo{
							Server:     srv,
							FullMethod: "/helloworld.Greeter/SayHello",
						}

						handler := func(ctx context.Context, req interface{}) (interface{}, error) {
							return reflect.ValueOf(srv).Convert(typ).MethodByName(e.Name).Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(req).Convert(inType)}), nil
						}

						return interceptor(ctx, in, info, handler)
					},
				})
			}

			endpoints = append(endpoints, e)
		} else {
			fmt.Println("Error is here ", e)
		}
	}

	nt := hdlr.Elem().Field(0).Type()

	return &rpcHandler{
		name:      name,
		handler:   handler,
		endpoints: endpoints,
		serviceDesc: grpc.ServiceDesc{
			ServiceName: name,
			HandlerType: reflect.New(nt).Interface(),
			Methods:     methods,
			Streams:     streams,
			Metadata:    "",
		},
		opts: options,
	}
}

func (r *rpcHandler) Name() string {
	return r.name
}

func (r *rpcHandler) Handler() interface{} {
	return r.handler
}

func (r *rpcHandler) Endpoints() []*registry.Endpoint {
	return r.endpoints
}

func (r *rpcHandler) Options() server.HandlerOptions {
	return r.opts
}

func (r *rpcHandler) getServiceDesc() *grpc.ServiceDesc {
	fmt.Println(r.serviceDesc)
	return &r.serviceDesc
}
