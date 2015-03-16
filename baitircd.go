package main

import (
    "bufio"
    "flag"
    "fmt"
    "log"
    "math/rand"
    "net"
    "runtime"
    "strings"
    "sync"
    "time"
)

const (
    letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890"
    modes   = "vhoaq"
)

var (
    maxChans   int
    maxQueries int
    port       string
)

type User struct {
    sync.Mutex
    nick     string
    host     string
    channels []Channel
    conn     net.Conn
    reader   *bufio.Reader
    sentQ    int
}

type Channel struct {
    c    []string
    name string
}

func findChannel(channels []Channel, name string) int {
    for i, n := range channels {
        if name[1:] == n.name { // We remove the # in the name
            return i
        }
    }

    return -1
}

func randS(length int, spaces bool) string {
    length -= rand.Intn(length) - 1
    str := make([]byte, length)

    set := []byte(letters)
    if spaces {
        set = append(set, ' ')
    }

    for i := range str {
        str[i] = set[rand.Intn(len(set))]
    }

    return string(str)
}

func (u *User) send(message string, args ...interface{}) {
    u.conn.SetWriteDeadline(time.Now().Add(time.Minute))

    _, err := u.conn.Write([]byte(fmt.Sprintf(message+"\n", args...)))
    if err != nil {
        panic(err)
    }
}

func (u *User) selfJoin() {
    channel := Channel{name: randS(10, false)}

    u.send(":%s!%s JOIN :#%s", u.nick, u.host, channel.name)
    u.send(":irc.bait.rekt 353 %s = #%s :%s", u.nick, channel.name, u.nick)
    u.send(":irc.bait.rekt 366 %s #%s :End of NAMES list", u.nick, channel.name)

    u.channels = append(u.channels, channel)
}

func (u *User) selfMsg() {
    message := randS(200, true)
    if rand.Intn(20) == 0 {
        message += " " + u.nick + " "
    }

    u.send(":%s!%s PRIVMSG %s :%s",
        randS(15, false), randS(30, false), u.nick, message)
}

func (u *User) chanJoin(channel *Channel) {
    nick := randS(15, false)

    channel.c = append(channel.c, nick)

    u.send(":%s!%s JOIN :#%s",
        nick, randS(30, false), channel.name)
}

func (u *User) chanPart(channel *Channel) {
    if len(channel.c) < 2 {
        return
    }

    var nick string
    nick, channel.c = channel.c[0], channel.c[1:]

    u.send(":%s!%s PART #%s :%s",
        nick, randS(30, false), channel.name, randS(30, true))
}

func (u *User) chanQuit(channel *Channel) {
    if len(channel.c) < 2 {
        return
    }

    var nick string
    nick, channel.c = channel.c[0], channel.c[1:]

    u.send(":%s!%s QUIT :%s",
        nick, randS(30, false), randS(30, true))
}

func (u *User) chanKick(channel *Channel) {
    if len(channel.c) < 3 {
        return
    }

    var nick1, nick2 string
    nick1, nick2, channel.c = channel.c[0], channel.c[1], channel.c[1:]

    u.send(":%s!%s KICK #%s %s :%s",
        nick2, randS(30, false), channel.name, nick1, randS(30, true))
}

func (u *User) chanMsg(channel *Channel) {
    if len(channel.c) < 1 {
        return
    }

    nick := channel.c[rand.Intn(len(channel.c))]
    message := randS(200, true)
    if rand.Intn(20) == 0 {
        message += " " + u.nick + " "
    }

    u.send(":%s!%s PRIVMSG #%s :%s",
        nick, randS(30, false), channel.name, message)
}

func (u *User) chanTopic(channel *Channel) {
    if len(channel.c) < 1 {
        return
    }

    nick := channel.c[rand.Intn(len(channel.c))]

    u.send(":%s!%s TOPIC #%s :%s",
        nick, randS(30, false), channel.name, randS(60, true))
}

func (u *User) chanMode(channel *Channel) {
    if len(channel.c) < 2 {
        return
    }

    n := rand.Intn(len(channel.c) - 1)
    nick1, nick2 := channel.c[n], channel.c[n+1]
    mode := modes[rand.Intn(len(modes))]

    action := '+'
    if rand.Intn(5) == 0 {
        action = '-'
    }

    u.send(":%s!%s MODE #%s %c%c %s",
        nick1, randS(30, false), channel.name, action, mode, nick2)
}

func (u *User) chanNick(channel *Channel) {
    if len(channel.c) < 1 {
        return
    }

    nick := &channel.c[rand.Intn(len(channel.c))]
    newNick := randS(15, false)

    u.send(":%s!%s NICK :%s",
        *nick, randS(30, false), newNick)

    *nick = newNick
}

func (u *User) handle() {
    defer func() { // We use this to exit if the socket is closed deep down
        if r := recover(); r != nil {
            if _, ok := r.(runtime.Error); ok { // Real panic, above our pay grade
                panic(r)
            }
            log.Println(r)
        }
    }()

    log.Println("Received connection from", u.host, "!")

    u.reader = bufio.NewReader(u.conn)

    for u.nick == "" {
        u.conn.SetDeadline(time.Now().Add(time.Minute))
        line, err := u.reader.ReadString('\n')
        if err != nil {
            log.Println(err)
            u.conn.Close()
            return
        }

        if strings.HasPrefix(line, "NICK") {
            u.nick = line[5 : len(line)-1]
            log.Println("Nick is: " + u.nick)
        }
    }

    defer log.Println("Rip: ", u.nick+"@"+u.host)

    go func() {
        for {
            u.conn.SetDeadline(time.Time{})
            line, err := u.reader.ReadString('\n')
            if err != nil {
                log.Println(err)
                u.conn.Close()
                return
            }

            line = strings.TrimRight(line, "\n")

            lineS := strings.Split(line, " ")
            command := lineS[0]

            switch command {
            case "PING":
                u.send("PONG " + line[5:])

            case "QUIT":
                u.conn.Close()
                log.Println("QUIT")
                return

            case "NICK":
                if len(lineS) >= 2 {
                    newNick := lineS[1]
                    log.Println("New nick for", u.nick, "is:", newNick)
                    u.nick = newNick
                }

            case "MODE":
                if len(lineS) >= 2 {
                    u.send(":irc.bait.rekt 324 %s +", lineS[1])
                }

            case "PART":
                if len(lineS) >= 2 {
                    if n := findChannel(u.channels, lineS[1]); n > 0 {
                        u.Lock()

                        if n == len(u.channels)-1 {
                            u.channels = u.channels[:n]
                        } else {
                            u.channels = append(u.channels[:n], u.channels[n+1:]...)
                        }

                        u.Unlock()
                    }
                }
            }
        }
    }()

    u.send(":irc.bait.rekt 001 %s :Hi", u.nick) // Bunch of default welcome messages
    u.send(":irc.bait.rekt 002 %s :Pls", u.nick)
    u.send(":irc.bait.rekt 003 %s :Bye", u.nick)
    u.send(":irc.bait.rekt 004 %s :Hi", u.nick)
    u.send(":irc.bait.rekt 005 %s PREFIX=(qaohv)~&@%%+", u.nick)

    start := time.Now()

    for {
        if time.Since(start) >= time.Minute {
            time.Sleep(time.Second * 5)
            start = time.Now()
        }

        u.Lock()

        if n := rand.Intn(10); n == 0 && len(u.channels) < maxChans {
            u.selfJoin()
        } else if n == 1 && len(u.channels) == maxChans && u.sentQ < maxQueries {
            u.selfMsg()
            u.sentQ++
        }

        if len(u.channels) < maxChans {
            u.Unlock()
            continue
        }

        for i := range u.channels {
            switch j := rand.Intn(25); {
            case j <= 5:
                u.chanJoin(&u.channels[i])

            case j == 11:
                u.chanPart(&u.channels[i])

            case j == 12:
                u.chanQuit(&u.channels[i])

            case j == 13:
                u.chanKick(&u.channels[i])

            case j == 14:
                u.chanTopic(&u.channels[i])

            case j == 15:
                u.chanMode(&u.channels[i])

            case j == 16:
                u.chanNick(&u.channels[i])

            default:
                u.chanMsg(&u.channels[i])
            }
        }

        u.Unlock()
    }
}

func main() {
    rand.Seed(time.Now().UnixNano())

    flag.IntVar(&maxChans, "c", 100, "Maximum channels number")
    flag.IntVar(&maxQueries, "q", 100, "Maximum queries the client will receive")
    flag.StringVar(&port, "p", "8888", "Port to use")
    flag.Parse()

    ln, err := net.Listen("tcp", ":"+port)
    if err != nil {
        log.Fatalln(err)
    }

    runtime.GOMAXPROCS(runtime.NumCPU())

    for {
        conn, err := ln.Accept()
        if err != nil {
            log.Println(err)
            continue
        }

        newUser := new(User)
        newUser.conn = conn
        newUser.host = conn.RemoteAddr().String()
        go newUser.handle()
    }
}
