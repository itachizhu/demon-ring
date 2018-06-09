package inet

import (
	"net"
	"time"
	"sync"
	"context"
	"golang.org/x/net/trace"
	"math"
	"errors"
	"runtime"
	"fmt"
)

const (
	defaultServerMaxReceiveMessageSize = 1024 * 1024 * 4
	defaultServerMaxSendMessageSize    = math.MaxInt32
)

type options struct {
	maxConcurrentStreams  uint32
	maxReceiveMessageSize int
	maxSendMessageSize    int
	initialWindowSize     int32
	initialConnWindowSize int32
	writeBufferSize       int
	readBufferSize        int
}

var defaultServerOptions = options {
	maxReceiveMessageSize: defaultServerMaxReceiveMessageSize,
	maxSendMessageSize:    defaultServerMaxSendMessageSize,
}

type Connection struct {
	conn net.Listener
	server Server
	userId uint64
}

type Server struct {
	opts options
	mu     sync.Mutex // guards following
	lis    map[net.Listener]bool
	connections map[net.Listener]Connection
	serve  bool
	drain  bool
	ctx    context.Context
	cancel context.CancelFunc
	// A CondVar to let GracefulStop() blocks until all the pending RPCs are finished
	// and all the transport goes away.
	cv     *sync.Cond
	events trace.EventLog
}

type ServerOption func(*options)

func WriteBufferSize(s int) ServerOption {
	return func(o *options) {
		o.writeBufferSize = s
	}
}

func ReadBufferSize(s int) ServerOption {
	return func(o *options) {
		o.readBufferSize = s
	}
}

func InitialWindowSize(s int32) ServerOption {
	return func(o *options) {
		o.initialWindowSize = s
	}
}

func InitialConnWindowSize(s int32) ServerOption {
	return func(o *options) {
		o.initialConnWindowSize = s
	}
}

func MaxMsgSize(m int) ServerOption {
	return MaxRecvMsgSize(m)
}

func MaxRecvMsgSize(m int) ServerOption {
	return func(o *options) {
		o.maxReceiveMessageSize = m
	}
}

func MaxSendMsgSize(m int) ServerOption {
	return func(o *options) {
		o.maxSendMessageSize = m
	}
}

func MaxConcurrentStreams(n uint32) ServerOption {
	return func(o *options) {
		o.maxConcurrentStreams = n
	}
}

type ConnectHandler interface {
	handleConnect(conn net.Conn)
}

var (
	// ErrServerStopped indicates that the operation is now illegal because of
	// the server being stopped.
	ErrServerStopped = errors.New("demon-ring: the server has been stopped")
)

func NewServer(opt ...ServerOption) *Server {
	opts := defaultServerOptions
	for _, o := range opt {
		o(&opts)
	}
	s := &Server{
		lis:   make(map[net.Listener]bool),
		opts:  opts,
		connections: make(map[net.Listener]Connection),
	}
	s.cv = sync.NewCond(&s.mu)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	_, file, line, _ := runtime.Caller(1)
	s.events = trace.NewEventLog("demon-ring.Server", fmt.Sprintf("%s:%d", file, line))
	return s
}

func (s *Server) ServeTCP(port string, handler ConnectHandler) {
	defer func() {
		if err := recover(); err != nil {
			// 处理全局异常
		}
	}()

	ln, err := net.Listen("tcp", port)
	if err != nil {
		// 处理全局异常
	}
	defer ln.Close()

	// 配置连接保持属性
	if tcpl, ok :=  ln.(*net.TCPListener); ok {
		// Wrap TCP listener to enable TCP keep-alive
		ln, err := tcpl.AcceptTCP()
		if err != nil {
			// 处理异常
			return
		}
		ln.SetKeepAlive(true)
		ln.SetKeepAlivePeriod(30 * time.Second)
	}

	var tempDelay time.Duration
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			return
		}
		// 启动新的chan处理客户端连接
		go handler.handleConnect(conn)
	}
}

func (s *Server) Serve(lis net.Listener) error {
	s.mu.Lock()
	//s.printf("serving")
	s.serve = true
	if s.lis == nil {
		s.mu.Unlock()
		lis.Close()
		return ErrServerStopped
	}
	s.lis[lis] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		if s.lis != nil && s.lis[lis] {
			lis.Close()
			delete(s.lis, lis)
		}
		s.mu.Unlock()
	}()

	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		rawConn, err := lis.Accept()
		if err != nil {
			if ne, ok := err.(interface {
				Temporary() bool
			}); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				//s.mu.Lock()
				//s.printf("Accept error: %v; retrying in %v", err, tempDelay)
				//s.mu.Unlock()
				timer := time.NewTimer(tempDelay)
				select {
				case <-timer.C:
				case <-s.ctx.Done():
				}
				timer.Stop()
				continue
			}
			//s.mu.Lock()
			//s.printf("done serving; Accept = %v", err)
			//s.mu.Unlock()
			return err
		}
		tempDelay = 0
		// Start a new goroutine to deal with rawConn
		// so we don't stall this Accept loop goroutine.
		go s.handleRawConn(rawConn)
	}
}

func (s *Server) handleRawConn(rawConn net.Conn) {

}