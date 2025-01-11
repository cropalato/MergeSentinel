# Running dev gitlab docker

```bash
IP=$(ip r|head -n1|sed 's/^.*src \([^ ]*\) .*/\1/')
echo -e "${IP}\tgitlab.cropa.ca" | sudo tee -a /etc/hosts
docker run -d --rm --hostname gitlab.cropa.ca --env GITLAB_OMNIBUS_CONFIG="external_url='http://gitlab.cropa.ca'" --env "GITLAB_ROOT_PASSWORD=wqerty123." --publish 443:443 --publish 80:80 --name gitlab gitlab/gitlab-ce
```

# Configuring gitlab

- Create a personal access token in root profile using http://gitlab.acme.co
- Create projects and users

# Running command

```bash
go run main.go -url ${GITLAB_URL} -token ${GITLAB_API_KEY}
```
