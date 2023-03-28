# gowebstats

A web stats gatherer in [go](https://go.dev) written using https://chat.openai.com.

The purpose is to run this service (say running on https://mystats.mydomain.com/
collect visitor web stats for websites you control.

You include a reference this server as a "CSS" like this in the pages you want 
to track:

	<link href=https://mystats.mydomain.com/hello.css rel=stylesheet>

The request will be logged into a JSON file. Processing that JSON file as a 
separate process to collect user visit stats.

*Note:* The `hello.css` is just a subterfuge. All requests to the server return 
an empty CSS.

**The ChatGPT-3.5 prompt**

> write a go http server program that responds to all requests with an empty css file with correct http headers, and saves the request info - time, ip, and user agent into a queue of size 1000, and when the queue gets full, it is written to a timestamped json file under a user specified location from config.toml; the http server should only log requests made to whitelisted  domains; the whitelisted domains are read from a config.toml file. the config.toml is accepted as a command line argument --config, with the default value being config.toml


The ChatGPT Response is the go file.

---

## Deployment

After the generation of `gowebstats.go` above, I ran the following:

	go mod init btbytes.com/gowebstats
	go mod tidy
	go build

Copied this `config.toml` from [sweetpywebstats](https://github.com/btbytes/sweetpywebstats) and modified the config keys to `LogDir` and `WhitelistedDomains` to match Go style `CapitalCase`.

