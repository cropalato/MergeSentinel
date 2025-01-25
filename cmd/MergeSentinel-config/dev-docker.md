# Running dev gitlab docker

```bash
IP=$(ip r|head -n1|sed 's/^.*src \([^ ]*\) .*/\1/')
echo -e "${IP}\tgitlab.acme.co" | sudo tee -a /etc/hosts
docker run -d --rm --hostname gitlab.acme.co --env GITLAB_OMNIBUS_CONFIG="external_url='http://gitlab.acme.co'" --env "GITLAB_ROOT_PASSWORD=wqerty123." --publish 443:443 --publish 80:80 --name gitlab gitlab/gitlab-ce
```

# Configuring gitlab

- Create a personal access token in root profile using http://gitlab.acme.co
- Create some projects and users
- Create a config.json

```bash
export GITLAB_API_KEY=<new created key>
echo "{ \"gitlab_token\": \"${GITLAB_API_KEY}\", \"gitlab_url\": \"http://gitlab.acme.co\", \"webhook_token\": \"\", \"PgresConn\": \"\", \"projects\": [{}]}" | jq -M | tee config.json
```

# Running command

```bash
go run main.go
```
