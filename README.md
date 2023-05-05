labelflared
===========

This tool assists in operating cloudflared within a standalone docker or docker-compose environment.
It enables the definition of cloudflared ingress rules using container labels.

Example docker-compose.yml
==========================

```yml
version: '3'
services:
    cloudflared:
        container_name: cloudflared
        image: cloudflare/cloudflared:2023.4.2
        volumes:
            - "cloudflared_config:/etc/cloudflared"
        command: tunnel --no-autoupdate run
        labels:
            - "labelflared.cloudflared"
        depends_on:
            labelflared:
                condition: service_healthy
        restart: on-failure
    labelflared:
        container_name: labelflared
        image: labelflared
        volumes:
            - "cloudflared_config:/etc/cloudflared"
            - "/var/run/docker.sock:/var/run/docker.sock"
        healthcheck:
            test: ["CMD", "test", "-s", "/etc/cloudflared/config.yml"]
            interval: '5s'
        environment:
            CLOUDFLARED_TOKEN: "<b64 encoded cloudflared token>"
        restart: on-failure
    vaultwarden:
        container_name: vaultwarden
        image: vaultwarden/server:1.27.0
        volumes:
            - "/data/vaultwarden/data:/data"
        environment:
            SIGNUPS_ALLOWED: "false"
            WEBSOCKET_ENABLED: "true"
            SMTP_HOST: "smtp.example.com"
            SMTP_PORT: "465"
            SMTP_SECURITY: "force_tls"
            SMTP_FROM: "bitwarden@example.com"
            DOMAIN: "https://bitwarden.example.com"
            ADMIN_TOKEN: "<...>"
        restart: on-failure
        labels:
            - "labelflared.ingress.vaultwarden-websocket.hostname=bitwarden.example.com"
            - "labelflared.ingress.vaultwarden-websocket.port=3012"
            - "labelflared.ingress.vaultwarden-websocket.path=/notifications/hub"
            - "labelflared.ingress.vaultwarden-websocket.priority=1000"
            - "labelflared.ingress.vaultwarden-web.hostname=bitwarden.example.com"
            - "labelflared.ingress.vaultwarden-web.port=80"

volumes:
    cloudflared_config:
        name: "cloudflared_config"
```

Environment Variables
=====================

- `CLOUDFLARED_TOKEN` - Base64 encoded cloudflared token. Required.
- `CLOUDFLARED_CONFIG_DIR` - Path to cloudflared config directory. Defaults to `/etc/cloudflared`.
- `LABEL_PREFIX` - Initial segment of a label that you can alter to form distinct sets of containers and their corresponding cloudflared instances. Defaults to `labelflared`.
- `SETTLEDOWN_TIMEOUT` - time in seconds to wait for things to stop changing before updating cloudflared configuration. Defaults to 5.

Label Syntax
============

cloudflared container must have `labelflared.cloudflared` label.

Service containers can have one or multiple ingress rules defined:
- `labelflared.ingress.<rule_name>.protocol` - defaults to `http`
- `labelflared.ingress.<rule_name>.hostname` - required if no `path` is set
- `labelflared.ingress.<rule_name>.port` - defaults to `80`
- `labelflared.ingress.<rule_name>.path` - required if no `hostname` is set
- `labelflared.ingress.<rule_name>.priority` - the higher the number, the greater the rule's priority. defaults to `0`