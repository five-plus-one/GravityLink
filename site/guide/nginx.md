# 域名与 HTTPS

GravityLink 不签发证书、不终止 TLS。用 Nginx（或 Caddy、Traefik 等）做 HTTPS 与路由。

## DNS

| 记录 | 主机名 | 值 | 用途 |
|------|--------|-----|------|
| A | `s` | 服务器 IP | 入口短链 |
| A | `page` | 服务器 IP | 落地页（可选） |
| A | `admin` | 服务器 IP | 管理后台（建议限制） |

解析生效后再到后台添加对应域名。

## 短链 / 落地反代

```nginx
server {
    listen 443 ssl http2;
    server_name s.yourdomain.com page.yourdomain.com;

    ssl_certificate     /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;

    client_max_body_size 10m;  # 若有素材上传，经后台反代时需要

    location / {
        proxy_pass http://127.0.0.1:18080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 80;
    server_name s.yourdomain.com page.yourdomain.com;
    return 301 https://$host$request_uri;
}
```

**`Host` 必须原样传**，否则 GravityLink 认不出域名类型。

## 管理后台反代

```nginx
server {
    listen 443 ssl http2;
    server_name admin.yourdomain.com;

    ssl_certificate     /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;

    # 强烈建议：办公网 / VPN IP
    # allow 203.0.113.10;
    # deny all;

    location / {
        proxy_pass http://127.0.0.1:18081;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 证书

用 acme.sh / certbot 签 Let’s Encrypt 后：

```bash
sudo nginx -t
sudo nginx -s reload
```

## 后台「公开访问地址」

初始化或设置里的公开地址，应与用户真正打开的入口一致：

```text
https://s.yourdomain.com
```

不要填成管理后台域名，也不要填成未解析的临时 IP（除非只做本机测试）。

## 多入口域名

后台可添加多个 entry。Nginx 给每个域名都写一条 `server`，都反代到 18080 即可。链接创建时选择用哪个入口。

## 自检清单

- [ ] `curl -I https://s.yourdomain.com/某个短码` 能 302/301 而不是 404  
- [ ] 响应头里 Location 是预期目标或落地域  
- [ ] 后台仅白名单可访问  
- [ ] 短链端口 18080 不对公网裸奔（只给 Nginx 回源）  
