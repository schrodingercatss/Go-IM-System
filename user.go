package main

import (
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C chan string
	conn net.Conn
	server *Server
}

// 创建一个用户的API
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name: userAddr,
		Addr: userAddr,
		C: make(chan string),
		conn: conn,
		server: server,
	}
	go user.ListenMessage() // 启动监听当前User channel的goroutine
	return user
}

// 监听当前User channel的方法，一旦有消息，就直接发送给对端客户端
func (u *User) ListenMessage() {
	for {
		msg := <- u.C
		u.conn.Write([]byte(msg + "\n"))
	}
}

// 用户的上线业务
func (u *User) Online() {
	// 用户上线，将用户加入onlineMap中
	u.server.mapLock.Lock()
	u.server.OnlineMap[u.Name] = u
	u.server.mapLock.Unlock()

	// 广播当前用户上线的消息
	u.server.BroadCast(u, "已上线")
}

// 用户的下线业务
func (u *User) Offline() {
	// 用户下线，将用户移出onlineMap中
	u.server.mapLock.Lock()
	delete(u.server.OnlineMap, u.Name)
	u.server.mapLock.Unlock()

	// 广播当前用户上线的消息
	u.server.BroadCast(u, "已下线")
}

// 用户处理消息的业务
func (u *User) DoMessage(msg string) {
	if msg == "who" {
		// 查询当前在线用户都有哪些人
		u.getALlOnlineUser()
	} else if len(msg) > 7 && msg[:7] == "rename|"{
		u.changeUserName(strings.Split(msg, "|")[1])
	} else if len(msg) > 2 && msg[:3] == "to|" {
		// 消息格式： to|userName|msg
		u.sendPrivateMsg(msg)
	} else {
		u.server.BroadCast(u, msg)
	}
}

// 给当前user对应的客户端发送消息
func (u *User) SendMsg(msg string) {
	u.conn.Write([]byte(msg))
}

// 查询在线用户
func (u *User) getALlOnlineUser() {
	u.server.mapLock.Lock()
	for _, user := range u.server.OnlineMap {
		onlineMsg := "[" + user.Addr + "]" + user.Name + "：" + "在线...\n"
		u.SendMsg(onlineMsg)
	}
	u.server.mapLock.Unlock()
}

// 修改用户名
func (u *User) changeUserName(newName string) {
	_, ok := u.server.OnlineMap[newName]
	if ok {
		u.SendMsg("当前用户名被使用\n")
	} else {
		u.server.mapLock.Lock()
		delete(u.server.OnlineMap, u.Name)
		u.server.OnlineMap[newName] = u
		u.server.mapLock.Unlock()
		u.Name = newName
		u.SendMsg("您的用户名已经更新为: " + u.Name + "\n")
	}
}

// 私聊用户
func (u *User) sendPrivateMsg(msg string) {
	// 1. 获取对方用户名
	remoteName := strings.Split(msg, "|")[1]
	if remoteName == "" {
		u.SendMsg("消息格式不正确，请使用\"to|张三|msg\"的格式\n")
		return
	}
	// 2. 根据用户名得到用户对象
	remoteUser, ok := u.server.OnlineMap[remoteName]
	if !ok {
		u.SendMsg("该用户不在线或不存在!\n")
		return
	}
	// 3. 获取消息内容，通过对方User将内容发送出去
	remoteUser.SendMsg(u.Name + "对您说: " + strings.Split(msg, "|")[2] + "\n")
}
