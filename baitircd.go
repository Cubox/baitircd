package main

import "flag"
import "fmt"
import "net"
import "bufio"
import "strings"
import "runtime"
import "time"
import "math/rand"

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890")
var maxChans int
var port string

type User struct {
    nick string
    host string
    channels []Channel
    conn net.Conn
    reader *bufio.Reader
}

type Channel struct {
    c []string
    name string
}

func randS(length int, spaces bool) string {
    length -= rand.Intn(length) - 1
    str := make([]byte, length)

    set := letters
    if spaces {
        set = append(set, ' ')
    }

    for i := range str {
        str[i] = set[rand.Intn(len(set))]
    }

    return string(str)
}

func (user *User) send(message string, args ...interface{}) {
    user.conn.SetWriteDeadline(time.Now().Add(time.Minute))

    _, err := user.conn.Write([]byte(fmt.Sprintf(message + "\n", args...)))
    if err != nil {
        panic(err)
    }
}

func (user *User) selfJoin() {
    channel := Channel{name: randS(10, false)}

    user.send(":%s!%s JOIN :#%s\n", user.nick, user.host, channel.name)
    user.send(":irc.bait.rekt 353 %s = #%s :%s\n", user.nick, channel.name, user.nick)
    user.send(":irc.bait.rekt 366 %s #%s :End of NAMES list\n", user.nick, channel.name)

    user.channels = append(user.channels, channel)
}

func (user *User) selfMsg() {
    user.send(":%s!%s PRIVMSG %s :%s\n",
        randS(15, false), randS(30, false), user.nick, randS(200, false))
}

func (user *User) chanJoin(channel *Channel) {
    nick := randS(15, false)

    channel.c = append(channel.c, nick)

    user.send(":%s!%s JOIN :#%s\n",
        nick, randS(30, false), channel.name)
}

func (user *User) chanPart(channel *Channel) {
    if len(channel.c) < 2 {
        return
    }

    var nick string
    nick, channel.c = channel.c[0], channel.c[1:]

    user.send(":%s!%s PART #%s :%s\n",
        nick, randS(30, false), channel.name, randS(30, true))
}

func (user *User) chanQuit(channel *Channel) {
    if len(channel.c) < 2 {
        return
    }

    var nick string
    nick, channel.c = channel.c[0], channel.c[1:]

    user.send(":%s!%s QUIT :%s\n",
        nick, randS(30, false), randS(30, true))
}

func (user *User) chanKick(channel *Channel) {
    if len(channel.c) < 3 {
        return
    }

    var nick1, nick2 string
    nick1, nick2, channel.c = channel.c[0], channel.c[1], channel.c[1:]

    user.send(":%s!%s KICK #%s %s :%s\n",
        nick2, randS(30, false), channel.name, nick1, randS(30, true))
}

func (user *User) chanMsg(channel *Channel) {
    if len(channel.c) < 1 {
        return
    }

    nick := channel.c[rand.Intn(len(channel.c))]

    user.send(":%s!%s PRIVMSG #%s :%s\n",
        nick, randS(30, false), channel.name, randS(100, false))
}

func (user *User) handle() {
    defer func() { // We use this to exit if the socket is closed deep down
        if r := recover(); r != nil {
            if _, ok := r.(runtime.Error); ok { // Real panic, above our pay grade
                panic(r)
            }
            fmt.Println(r)
        }
    }()

    fmt.Println("Received connection!")

    user.reader = bufio.NewReader(user.conn)

    for user.nick == "" {
        user.conn.SetDeadline(time.Now().Add(time.Minute))
        line, err := user.reader.ReadString('\n')
        if err != nil {
            fmt.Println(err)
            user.conn.Close()
            return
        }

        if strings.HasPrefix(line, "NICK") {
            user.nick = line[5 : len(line)-1]
            fmt.Println("Nick is: " + user.nick)
        }
    }

    defer fmt.Println("Rip :", user.nick)

    go func() {
        for {
            user.conn.SetDeadline(time.Time{})
            line, err := user.reader.ReadString('\n')
            if err != nil {
                fmt.Println(err)
                user.conn.Close()
                return
            }

            if strings.HasPrefix(line, "PING ") {
                user.send("PONG " + line[5:len(line)-1])

            } else if strings.HasPrefix(line, "QUIT ") {
                user.conn.Close()
                fmt.Println("QUIT")
                return
            }
        }
    }()

    user.send(":irc.bait.rekt 001 %s :Hi", user.nick) // Bunch of default welcome messages
    user.send(":irc.bait.rekt 002 %s :Pls", user.nick)
    user.send(":irc.bait.rekt 003 %s :Bye", user.nick)
    user.send(":irc.bait.rekt 004 %s :Hi", user.nick)

    for {
        if (rand.Intn(1) == 0 && len(user.channels) < maxChans) {
            user.selfJoin()
        } else {
            user.selfMsg()
        }

        for i := range user.channels {
            switch j := rand.Intn(50); {
            case j <= 10:
                user.chanJoin(&user.channels[i])

            case j == 11:
                user.chanPart(&user.channels[i])

            case j == 12:
                user.chanQuit(&user.channels[i])

            case j == 13:
                user.chanKick(&user.channels[i])

            default:
                user.chanMsg(&user.channels[i])
            }
        }
    }
}

func main() {
    rand.Seed(time.Now().UnixNano())

    flag.IntVar(&maxChans, "c", 100, "Maximum channels number")
    flag.StringVar(&port, "p", "8888", "Port to use")
    flag.Parse()

    ln, err := net.Listen("tcp", ":" + port)
    if err != nil {
        panic(err)
    }

    runtime.GOMAXPROCS(runtime.NumCPU())

    for {
        conn, err := ln.Accept()
        if err != nil {
            fmt.Println(err)
            continue
        }

        newUser := new(User)
        newUser.conn = conn
        newUser.host = conn.RemoteAddr().String()
        go newUser.handle()
    }
}
