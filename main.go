package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
)

var usersDB = DataBaseUsers{usersDB: map[*net.Conn]string{}}

func SendErrorMsg(connection net.Conn, errorText string) error {
	errMsg := ErrorMessageData{ErrorText: errorText}
	errMsgData, err := json.Marshal(errMsg)
	if err != nil {
		return err
	}

	msg := MessageData{MessageTypeStatus: ErrorT, Data: errMsgData}
	msgData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = connection.Write(msgData)

	return err
}

func SendNotification(notification string) error {
	notificationMsg := NotificationData{NotificationMessage: notification}
	notificationMsgData, err := json.Marshal(notificationMsg)
	if err != nil {
		return err
	}

	msg := MessageData{MessageTypeStatus: Notification, Data: notificationMsgData}
	msgData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	for connPtr := range usersDB.usersDB {

		_, err = (*connPtr).Write(msgData)
		if err != nil {
			log.Printf("client(%s) did not receive a notification becouse(%s)\n",
				(*connPtr).RemoteAddr().String(), err.Error())
		}
	}

	return err
}

func ReSendMessage(userName string, dataBt []byte, connection net.Conn) error {
	TxtMsgData := TextMessageData{}
	MsgData := MessageData{}

	err := json.Unmarshal(dataBt, &TxtMsgData)
	if err != nil {
		return err
	}

	TxtMsgData.UserName = userName
	MsgData.MessageTypeStatus = SendMessageT

	if MsgData.Data, err = json.Marshal(TxtMsgData); err != nil {
		return err
	}

	msgData, err := json.Marshal(MsgData)
	if err != nil {
		return err
	}

	// usersDB.userDbMutex.Lock()
	// defer usersDB.userDbMutex.Unlock()

	for connPtr := range usersDB.usersDB {
		if *connPtr == connection {
			continue
		}

		_, err = (*connPtr).Write(msgData)
		if err != nil {
			log.Printf("client(%s) did not receive a message from client(%s) becouse(%s)\n",
				(*connPtr).RemoteAddr().String(), connection.RemoteAddr().String(), err.Error())
		}
	}

	return nil
}

func GetMessageData(connection net.Conn) ([]byte, error) {
	data := make([]byte, 8192)
	n, err := connection.Read(data)
	if err != nil {
		return nil, err
	}

	return data[:n], nil
}

func ClientWorker(connection net.Conn, channel chan struct{}) {
	UserName, stat, err := Validate(connection)
	if err != nil {
		if stat {
			if err := SendErrorMsg(connection, err.Error()); err != nil {
				log.Println(err.Error())
			}
		} else {
			log.Println(err.Error())
		}

		return
	}

	defer func() {
		log.Printf("client(%s) disconnected\n", connection.RemoteAddr().String())
		connection.Close()

		usersDB.userDbMutex.Lock()
		delete(usersDB.usersDB, &connection)
		usersDB.userDbMutex.Unlock()

		if err := SendNotification(fmt.Sprintf("user \"%s\" has left the chat", UserName)); err != nil {
			log.Println(err.Error())
		}

		log.Printf("max: %d / now: %d\n", cap(channel), len(channel))
	}()

	msgData := MessageData{}
	onlineUsersData := OnlineUsersData{}
	data := make([]byte, 8192)
	var n int

	for {
		n, err = connection.Read(data)
		if err != nil {
			log.Printf(err.Error())
			if err := SendErrorMsg(connection, "unknown server error"); err != nil {
				log.Println(err.Error())
			}
			return
		}

		if err = msgData.UnmarshalMsgData(data[:n]); err != nil {
			log.Printf(err.Error())
			_ = SendErrorMsg(connection, "failed to convert message")
			continue
		}

		switch msgData.MessageTypeStatus {
		case SendMessageT:
			if err = ReSendMessage(UserName, msgData.Data, connection); err != nil {
				log.Println(err.Error())

				if err = SendErrorMsg(connection, "unknown server error"); err != nil {
					log.Println(err.Error())
				}
			}
		case GetOnlineUsers:
			usersDB.userDbMutex.Lock()

			onlineUsersData.Count = len(usersDB.usersDB)
			onlineUsersData.UserNames = make([]string, 0, onlineUsersData.Count)

			// add all usernames to slice
			for _, v := range usersDB.usersDB {
				onlineUsersData.UserNames = append(onlineUsersData.UserNames, v)
			}
			usersDB.userDbMutex.Unlock()

			msgData.MessageTypeStatus = OnlineUsers
			if msgData.Data, err = json.Marshal(onlineUsersData); err != nil {
				log.Println(err.Error())

				if err := SendErrorMsg(connection, "unknown server error"); err != nil {
					log.Println(err.Error())
				}
				continue
			}

			if data, err = json.Marshal(msgData); err != nil {
				log.Println(err.Error())
				err := SendErrorMsg(connection, "unknown server error")
				if err != nil {
					log.Println(err.Error())
				}
				continue
			}

			if _, err = connection.Write(data); err != nil {
				log.Println(err.Error())
			}

		}
	}
}

func (msgData *MessageData) UnmarshalMsgData(dataBt []byte) error {
	if err := json.Unmarshal(dataBt, msgData); err != nil {
		return err
	}

	return nil
}

func (loginMsgData *LoginMessageData) UnmarshalLoginMsgData(dataBt []byte) error {
	if err := json.Unmarshal(dataBt, loginMsgData); err != nil {
		return err
	}

	return nil
}

// ValidateUserName use only in Validate function
func ValidateUserName(UserName string) bool {

	for _, found := range usersDB.usersDB {
		if strings.HasPrefix(found, UserName) {
			return false
		}
	}

	return true
}

func Validate(connection net.Conn) (string, bool, error) {
	msgData := MessageData{}
	loginMsgData := LoginMessageData{}

	btData, err := GetMessageData(connection)
	if err != nil {
		return "", false, err
	}

	if err = msgData.UnmarshalMsgData(btData); err != nil {
		return "", false, err
	}

	if msgData.MessageTypeStatus == OpenConnectT {
		if err = loginMsgData.UnmarshalLoginMsgData(msgData.Data); err != nil {
			return "", false, err
		}

		usersDB.userDbMutex.Lock()
		defer usersDB.userDbMutex.Unlock()

		if !ValidateUserName(loginMsgData.UserName) {
			return "", true, fmt.Errorf("user with this username is already logged in")
		}
	} else {
		return "", true, fmt.Errorf("invalid message format")
	}

	usersDB.usersDB[&connection] = loginMsgData.UserName
	return loginMsgData.UserName, true, nil
}

func main() {
	confFile, err := GetConfigFileData("config.json")
	if err != nil {
		log.Fatalln(err)
	}

	channels := make(chan struct{}, confFile.MaxConn)

	listener, err := net.Listen("tcp", confFile.Host+":"+confFile.Port)
	if err != nil {
		log.Fatalln(err)
	}

	defer listener.Close()

	log.Printf("Server is listening [%s:%s]\n", confFile.Host, confFile.Port)
	var connection net.Conn

	for {
		connection, err = listener.Accept()
		if err != nil {
			log.Println(err.Error())
			continue
		}

		select {
		case channels <- struct{}{}:
			go ClientWorker(connection, channels)
		default:
			if err = SendErrorMsg(connection,
				fmt.Sprintf("The maximum number of users has been reached on the server [%d/%d]",
					len(channels), cap(channels))); err != nil {
				log.Println(err.Error())
			}
			continue
		}

		log.Printf("client(%s) connected\n", connection.RemoteAddr().String())
		log.Printf("max: %d / now: %d\n", cap(channels), len(channels))
	}
}
