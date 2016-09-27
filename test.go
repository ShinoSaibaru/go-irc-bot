package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

type EventType int
type Mode int
type Action int

const (
	USUAL Mode = iota
	VOICE
	HALFOPERATOR
	OPERATOR
	ADMIN
)

const (
	MESSAGE EventType = iota
	PRIVATEMESSAGE
	JOIN
	PART
	QUIT
	NOTICE
	TITLECHANGED
)

const (
	SENDMESSAGE Action = iota
	SENDPRIVATEMESSAGE
	KICK
	BAN
)

type IRCPeople []IRCPerson

type LoadData struct {
	People       IRCPeople
	ChannelTitle string
}

type ActionData struct {
	FieldA string
	FieldB string
	FieldC string
}

type EventData struct {
	eventType EventType
	person    IRCPerson
	date      time.Time
	message   string
}

type IRCPerson struct {
	hostname string
	nick     string
	realName string
	mode     Mode
}

type ActionMap map[Action][]ActionData
type Callback func(eventData EventData) ActionMap
type Registrar func(eventName string, callback Callback)

type IRCBot struct {
	buffer     bufio.Reader
	connected  bool
	server     string
	port       int
	password   string
	connection net.Conn
	channel    string
	people     IRCPeople
	nick       string
	realName   string
	botError   error
	plugins    map[string]Plugin
	events     map[EventType][]Callback
}

func (instance *IRCBot) registerEvent(name EventType, callback Callback) {
	instance.events[name] = append(instance.events[name], callback)
}

func (instance *IRCBot) activatePlugin(name string) {
	instance.plugins[name].OnLoad()
}

func (instance *IRCBot) Error() error {
	return instance.botError
}

func CreateIRCBot(server, nick, realName, password string, port int, channel string) IRCBot {
	var instance IRCBot
	instance.server, instance.nick, instance.realName, instance.password = server, nick, realName, password
	instance.port = port
	instance.channel = channel
	instance.connected = false
	instance.plugins = make(map[string]Plugin)
	return instance
}

func (instance *IRCBot) Connect() error {
	if !instance.connected {
		instance.connection, instance.botError = net.Dial("tcp", instance.server+":"+strconv.Itoa(instance.port))
		if instance.botError != nil {
			fmt.Fprint(os.Stderr, instance.botError)
			return instance.botError
		}
		instance.buffer = bufio.NewReader(instance.connection)
		fmt.Fprint(instance.connection, "USER "+instance.nick+" 0 "+instance.nick+" :"+instance.realName+"\r\n")
		fmt.Fprint(instance.connection, "NICK "+instance.nick+"\r\n")
		fmt.Fprint(instance.connection, "JOIN "+instance.channel+"\r\n")
		instance.connected = true
		return nil
	}
	return errors.New("Not connected.")
}

func (instance *IRCBot) Join(channel string) error {
	if !instance.connected {
		return errors.New("Not connected.")
	}
	if err := instance.Leave(); err != nil {
		return err
	}
	_, instance.botError = fmt.Fprint(instance.connection, "JOIN :"+channel+"\r\n")
	if instance.botError != nil {
		return instance.botError
	}
	instance.channel = channel
	return nil
}

func (instance *IRCBot) Leave() error {
	if !instance.connected {
		return errors.New("Not connected.")
	}
	_, instance.botError = fmt.Fprint(instance.connection, "PART :"+instance.channel+"\r\n")
	return instance.botError
}

func (instance *IRCBot) Ping() error {
	if !instance.connected {
		return errors.New("Not connected.")
	}
	_, instance.botError = fmt.Fprint(instance.connection, "PING :PING\r\n")
	if instance.botError != nil {
		fmt.Fprint(os.Stderr, instance.botError)
		return instance.botError
	} else {
		return nil
	}
}

func (instance *IRCBot) pong(message string) error {
	if !instance.connected {
		return errors.New("Not connected.")
	}
	pongMessage := []byte(message)
	pongMessage[1] = 'O'
	_, instance.botError = fmt.Fprint(instance.connection, string(pongMessage))
	if instance.botError != nil {
		fmt.Fprint(os.Stderr, instance.botError)
		return instance.botError
	}
	return nil
}

func (instance *IRCBot) SendMessage(message, receiver string) error {
	if !instance.connected {
		return errors.New("Not connected.")
	}
	_, instance.botError = fmt.Fprint(instance.connection, "PRIVMSG "+receiver+" :"+message+"\r\n")
	if instance.botError != nil {
		fmt.Fprint(os.Stderr, instance.botError)
		return instance.botError
	}
}

func (instance *IRCBot) Quit() error {
	if !instance.connected {
		return errors.New("Not connected.")
	}
	instance.botError = instance.connection.Close()
	if instance.botError != nil {
		fmt.Fprint(os.Stderr, instance.botError)
		return instance.botError
	}
	instance.connected = false
	return nil
}

func (instance *IRCBot) receiveMessage() error {
	if !instance.connected {
		return errors.New("Not connected.")
	}
	var message string
	message, instance.botError = instance.buffer.ReadString('\n')
	if instance.botError != nil {
		return instance.botError
	}
	message += "\n"
	switch {
	case message[0:4] == "PING":
		instance.pong()
	}

	//	FIRE CALLBACKS HERE!!!!!!!!!!!!!!!!!!!!!!

	return nil
}

func (instance *IRCBot) RegisterPlugin(name string, plugin Plugin) {
	instance.plugins[name] = plugin
	instance.plugins[name].RegisterEvents(instance.registerEvent)
}

type Plugin interface {
	IsActive() bool
	RegisterEvents(registrar Registrar)
	OnLoad(loadData LoadData)
	OnUnload()
	PluginHelp() string
	PluginOptions(option string) error
}

type ExamplePlugin struct {
	isActive bool
	counter  int
	nigga    string
	options  [3]int
}

func (instance *ExamplePlugin) onJoin(eventData EventData) ActionMap {
	ret := make(ActionMap)
	ret[eventData.person] = "JOIN"
	return ret
}

func (instance *ExamplePlugin) onLeave(eventData EventData) ActionMap {
	ret := make(ActionMap)
	ret[eventData.person] = "LEAVE"
	return ret
}

func (instance *ExamplePlugin) onQuit(eventData EventData) ActionMap {
	ret := make(ActionMap)
	ret[eventData.person] = "QUIT"
	return ret
}

func (instance *ExamplePlugin) onMessage(eventData EventData) ActionMap {
	ret := make(ActionMap)
	ret[eventData.person] = "MESSAGE"
	return ret
}

func InitExamplePlugin() ExamplePlugin {
	var instance ExamplePlugin
	instance.counter = 1
	instance.nigga = ""
	instance.options = []int{9, 9, 9}
	instance.isActive = false
	return instance
}

func (instance *ExamplePlugin) RegisterEvents(registrar Registrar) {
	registrar(JOIN, instance.onJoin)
	registrar(LEAVE, instance.onLeave)
	registrar(QUIT, instance.onQuit)
	registrar(MESSAGE, instance.onMessage)
}

func (instance *ExamplePlugin) IsActive() bool {
	return instance.isActive
}

func (instance *ExamplePlugin) OnLoad() {
	instance.counter = 1
	instance.nigga = ""
	instance.options = []int{9, 9, 9}
	instance.isActive = true
}

func (instance *ExamplePlugin) PluginHelp() string {
	instance.nigga += "PluginHelp "
	return "lawl"
}

func (instance *ExamplePlugin) PluginOptions(option string) error {
	if option == "0" {
		instance.options[0] = 99
		return nil
	} else if option == "1" {
		instance.options[1] = 99
		return nil
	} else if option == "2" {
		instance.options[2] = 99
		return nil
	} else {
		return errors.New("yaYA!!")
	}
}

func (instance *ExamplePlugin) OnUnload() {
	fmt.Println("UNLOADED!!!!!!!!")
	instance.isActive = false
}

type ExamplePlugin1 struct {
	isActive bool
	counter  int
	nigga    string
	options  [3]int
}

func (instance *ExamplePlugin1) onJoin(eventData EventData) ActionMap {
	ret := make(ActionMap)
	ret[eventData.person] = "JOIN"
	return ret
}

func (instance *ExamplePlugin1) onLeave(eventData EventData) ActionMap {
	ret := make(ActionMap)
	ret[eventData.person] = "LEAVE"
	return ret
}

func (instance *ExamplePlugin1) onQuit(eventData EventData) ActionMap {
	ret := make(ActionMap)
	ret[eventData.person] = "QUIT"
	return ret
}

func (instance *ExamplePlugin1) onMessage(eventData EventData) ActionMap {
	ret := make(ActionMap)
	ret[eventData.person] = "MESSAGE"
	return ret
}

func InitExamplePlugin1() ExamplePlugin1 {
	var instance ExamplePlugin1
	instance.counter = 1
	instance.nigga = ""
	instance.options = []int{9, 9, 9}
	instance.isActive = false
	return instance
}

func (instance *ExamplePlugin1) RegisterEvents(registrar Registrar) {
	registrar(JOIN, instance.onJoin)
	registrar(LEAVE, instance.onLeave)
	registrar(QUIT, instance.onQuit)
	registrar(MESSAGE, instance.onMessage)
}

func (instance *ExamplePlugin1) IsActive() bool {
	return instance.isActive
}

func (instance *ExamplePlugin1) OnLoad() {
	instance.counter = 1
	instance.nigga = ""
	instance.options = []int{9, 9, 9}
	instance.isActive = true
}

func (instance *ExamplePlugin1) PluginHelp() string {
	instance.nigga += "PluginHelp "
	return "lawl"
}

func (instance *ExamplePlugin1) PluginOptions(option string) error {
	if option == "0" {
		instance.options[0] = 99
		return nil
	} else if option == "1" {
		instance.options[1] = 99
		return nil
	} else if option == "2" {
		instance.options[2] = 99
		return nil
	} else {
		return errors.New("yaYA!!")
	}
}

func (instance *ExamplePlugin1) OnUnload() {
	fmt.Println("UNLOADED!!!!!!!!")
	instance.isActive = false
}

//////////////////////////////////////////////////////////////////////////////////////////////

func main() {
	bot := CreateIRCBot("irc.furnet.org", "Kalota2", "Kalota2", 6667, "#gophers")
	if err := bot.Connect(); err != nil {
		panic(err)
	}
	bot.RegisterPlugin("yaYA", InitExamplePlugin())
	bot.RegisterPlugin("yaYA!!", InitExamplePlugin1())
}
