# helferlein

![Helferlein (Little Helper)](./media/helferlein.jpg)

> A small but helpful tool to set up a continuously-running git repository watcher to build CI/CD pipelines in seconds

## installation

Right now, there's no other way than to clone the repository and build the code by hand, but I'll add cross-compiled builds for all platforms and container images relatively soon!

## configuration

helferlein uses a YAML-based configuration to set up and manage repositories to watch and steps to run. An example configuration could look like this:

```yaml
# Fetch for updates every 10s
fetchInterval: "10s"

# Clone into .helferlein directory in current working directory
cloneDirectory: .helferlein

repositories:
  # Define "helferlein" repository to track
  - name: "helferlein"
    # Use GitHub https clone URL
    cloneUrl: "https://github.com/BrunoScheufler/helferlein.git"
    # Select "master" branch
    branches:
      - master
```

## authentication

Although public repositories can be cloned without credentials, helferlein requires to either use a username/password combination or an access token to clone and fetch contents from repositories.

Auth credentials can be added to the config or supplied as environment variables.

### adding credentials to configuration

```yaml
auth:
  # Either authenticate using access token
  accessToken: <token>

  # or using user/password
  user: <user>
  password: <password>
```

### supplying credentials as environment variables

```bash
# Add your access token like this
export HELFERLEIN_GIT_AUTH_ACCESS_TOKEN=<token>

# Or when using user/password
export HELFERLEIN_GIT_AUTH_USER=<user>
export HELFERLEIN_GIT_AUTH_PASSWORD=<password>
```

## usage

After configuring helferlein as explained in the steps above, you can simply run the executable which acceps the following options:

```bash
helferlein
  --loglevel [INFO]
  --config   [./config.yml]
```
