package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip string
	Port int

	//在线用户列表
	OnlineMap map[string] *User
	mapLock sync.RWMutex

	// 消息广播的channel
	Message chan string
}


// 创建一个server接口
func NewServer(ip string, port int) *Server {
	server := &Server {
		Ip: ip,
		Port: port,
		OnlineMap: make(map[string] *User),
		Message: make(chan string),
	}
	return server
}

// 广播消息的方法
func (s *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	s.Message <- sendMsg
}


func (s *Server) Handler(conn net.Conn) {
	// ... 当前连接的业务
	user := NewUser(conn, s)
	// 用户上线
	user.Online()

	// 监听用户是否活跃的channel
	isLive := make(chan bool)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err:", err)
				return
			}
			// 提取用户的消息（去除'\n'）
			msg := string(buf[:n - 1])

			// 用户针对msg进行消息处理
			user.DoMessage(msg)

			// 代表用户是活跃的
			isLive <-true
		}
	}()

	// 当前handler阻塞, 如果handler直接结束，那么goroutine也会直接结束
	for {
		select{
		case <-isLive:
			// 当前用户是活跃的，应该重置定时器
			// 不做任何事情，为了激活select，更新下面的定时器

		case <-time.After(time.Minute * 10):
			// 已经超时，将当前User强制关闭
			user.SendMsg("你被踢出群聊\n")
			user.Offline()
			//销毁资源
			close(user.C)

			//关闭链接
			conn.Close()

			//退出之前的haddler
			return
		}
	}
}

// 监听Message广播消息channel的goroutine，一旦有消息就发送给全部在线的User
func (s *Server) ListenMessager() {
	for {
		msg := <-s.Message

		// 将msg发送给全部在线User
		s.mapLock.Lock()
		for _, cli := range s.OnlineMap {
			cli.C <- msg
		}
		s.mapLock.Unlock()
	}
}


// 启动服务器的接口
func (s *Server) Start() {
	//socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Ip, s.Port))
	if err != nil {
		log.Fatal("net.Listen err:", err)
		return
	}
	// close listen socket
	defer listener.Close()

	// 启动监听Message的goroutine
	go s.ListenMessager()

	for {
		// accept
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("listener accept err:", err)
		}
		// do handler
		go s.Handler(conn)
	}
}



