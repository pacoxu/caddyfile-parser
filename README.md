# caddyfile-parser
Caddyfile Syntax https://caddyserver.com/docs/caddyfile

Nginx Conf & DNS corefile & Caddyfile

# Examples

## caddyfile example

```caddyfile
label1 {
	directive1 arg1
	directive2 arg2 {
	    subdir1 arg3 arg4
	    subdir2
	    # nested blocks not supported
	}
	directive3
}
```

## dns corefile example

```corefile
.:53 {
    errors
    health
    kubernetes cluster.local in-addr.arpa ip6.arpa {
       pods insecure
       upstream
       fallthrough in-addr.arpa ip6.arpa
    }
    prometheus :9153
    cache 30
}
```

## nginx conf example

```conf
user  root;
worker_processes  1;

error_log  /var/log/nginx/error.log warn;
pid        /var/run/nginx.pid;


events {
    worker_connections  65535;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;
    server_tokens off;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /var/log/nginx/access.log  main;

    sendfile        on;
    #tcp_nopush     on;

    keepalive_timeout  18400;


    proxy_connect_timeout       18400s;
    proxy_send_timeout          18400s;
    proxy_read_timeout          18400s;
    send_timeout                18400s;


    gzip  on;
    gzip_disable "msie6";

    client_max_body_size 0;

    include /etc/nginx/conf.d/*.conf;
}

```
