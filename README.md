# Authentication with Telegram in Go

I ran into issue that there's no end-to-end example, explaining how to log in with Telegram using Go, so I produced 
this nano-app; hopefully some of you will find it useful.

**Disclaimer:** this has not been vetted for the best security practices, and, in fact, was created as a toy
example, suitable for my own toy apps. Use at your own risk!

## Preparation

Before you start, I strongly encourage you to read [Telegram Login Widget](https://core.telegram.org/widgets/login) 
documentation, however limited it is. I strongly encourage you to understand it before proceeding.

Follow the steps at the Telegram page above to create your own Telegram bot which will be used for
authentication. Make sure to store bot token - you'll need it later!

Note, that when using `/setdomain` with Telegram bot it will refuse localhost domains - making testing
significantly harder. Furthermore, it clearly doesn't like custom ports (i.e. not 80/443). Here's how I 
worked around both limitations:

1. Set custom host in your system to point to `127.0.0.1`. On Linux/Mac it will be file named `/etc/hosts`;
on Windows it's `c:\windows\system32\drivers\etc\hosts`. Add an entry like: `127.0.0.1 tgauth.com` - I used domain
`tgauth.com` throughout this example, but you can choose something else.
2. Tricky bit - use `mitmproxy` to forward requests from a privileged port 80 to non-privileged port your test app
will be running (`8080` in this example). If you are using WSL as I did in this case, you will need to follow
the previous step _also_ in your WSL container as if you are on Linux (i.e. `/etc/hosts`). Then run:

    ```shell
    sudo mitmproxy --listen-host tgauth.com --listen-port 80 --mode reverse:http://localhost:8080/
    ```
    You will need to use `sudo` to use privileged port `80`.

If you've set up everything correctly, now all requests to [http://tgauth.com](http://tgauth.com) will be forwarded
to `localhost:8080`.

Finally, generate your login widget script code on the Telegram page mentioned above. Use your newly created
bot name, and select "Redirect to URL" option set to `http://tgauth.com/check_auth`. It will produce a snippet
of code like following:

```html
<script async src="https://telegram.org/js/telegram-widget.js?19" 
        data-telegram-login="sgzmd_tgauth_bot" 
        data-size="large" 
        data-auth-url="http://tgauth.com/check_auth" data-request-access="write"></script>
```

Now, take this code, open `main.go` and look for constant named `Html` - it also has a helpful `TODO`. Replace
the script tag in that constant with the snippet you've just produced.

## Running the example

Now this is the easy part. 

```shell
go run main.go -telegram_api_key=<your_bot_key>
```

Use the bot API key created in the step above. If everything works well, navigating to [http://tgauth.com](http://tgauth.com)
will produce a login page like this:

![Basic login page](/docassets/anon.png)

Follow the flow and pray everything works :)