package main

import (
	"net"
	"sync"
)

const (
	ErrorT       = 0
	Notification = 1

	OpenConnectT = 2
	SendMessageT = 3

	GetOnlineUsers = 4
	OnlineUsers    = 5
)

type DataBaseUsers struct {
	userDbMutex sync.Mutex
	usersDB     map[*net.Conn]string
}

type MessageData struct {
	MessageTypeStatus uint8  `json:"message_type_status"`
	Data              []byte `json:"data"`
}

type ErrorMessageData struct {
	ErrorText string `json:"error_text"`
}

type LoginMessageData struct {
	UserName string `json:"user_name"`
}

type TextMessageData struct {
	UserName string `json:"user_name"`
	TextData string `json:"text_data"`
}

type OnlineUsersData struct {
	Count     int      `json:"count"`
	UserNames []string `json:"user_names"`
}

type NotificationData struct {
	NotificationMessage string `json:"notification_message"`
}
