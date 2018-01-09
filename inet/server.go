package inet

import (
	"net"
	"time"
)

type ConnectHandler interface {
	handleConnect(conn net.Conn)
}

func ServeTCP(addr string, handler ConnectHandler) {
	defer func() {
		if err := recover(); err != nil {
			// 处理全局异常
		}
	}()

	ln, err := net.Listen("tcp", addr)
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