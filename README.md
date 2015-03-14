# baitircd

Small and fast tool to stress-test your irc client

## Why?

The inspiration for this is [https://github.com/weechat/weercd].

weercd is great, but lacks multi-client support, and can be buggy at times.

baitircd can handle as many clients as your processor can without melting! Very low memory and cpu usage!

Actually, the melting part will come from your irc client...

## How?

Install go, then clone this, then run `go run baitircd.go`

Nice, now start your irc client, and add localhost with port 8888.

For weechat:

```
weechat -d /tmp/test # the -d part will make your own irc client not die by spawning a new clean one
/server add test localhost/8888
/server connect test
```

And now you can pray.
