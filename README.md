# helferlein

![Helferlein (Little Helper)](./media/helferlein.jpg)

> A small but helpful tool to set up a continuously-running, branch-based Git repository watcher to build CI/CD pipelines in seconds

## features

- concurrent watching over as many projects as your system can handle
- branches are tracked completely independently, so you can handle multiple tracks (e.g. `main`, `staging`) 

## use cases

If you ever wanted to continuously watch git repository changes without a webhook setup or a complete CI/CD solution, this will be for you. I've created helferlein to simplify deployment workflows to my personal infrastructure, basically a self-built and self-hosted CI solution without all the additional bells and whistles.

## installation

Since I went ahead and cross-compiled helferlein for all systems imaginable, you can go ahead and download the OS- and architecture-specific binary from the [release page](https://github.com/BrunoScheufler/helferlein/releases). You can also just clone and build helferlein for yourself, the only important part is to build the `cmd` package (not to be confused with go's internal cmd package!), since this will boot everything up for you.

## configuration

helferlein uses a YAML-based [configuration](./worker/config.go) to set up and manage repositories to watch and steps to run. An example configuration could look like this:

```yaml
# Clone repositories into .helferlein directory
clone_directory: ".helferlein"
projects:
  helferlein:
    # Check for updates every 10 secnods
    fetch_interval: "10s"
    clone_url: "https://github.com/BrunoScheufler/helferlein.git"
    branches:
      main:
        steps:
          - echo "Hooray, changes! 🎉"
```

## authentication

While public repositories can be cloned without credentials, helferlein allows to supply use a username/password combination or an access token to clone and fetch contents from repositories with restricted access.

Auth credentials can be added to the project configuration or supplied as environment variables.

```yaml
projects:
  my_project:
    auth:
      username: user # or HELFERLEIN_GIT_AUTH_USER
      password: password # HELFERLEIN_GIT_AUTH_PASSWORD
```

```yaml
projects:
  my_project:
    auth:
      access_token: my_token # or HELFERLEIN_GIT_AUTH_ACCESS_TOKEN
```

## usage

After configuring helferlein as explained in the steps above, you can simply run the executable which acceps the following options:

```bash
helferlein
  --loglevel [INFO]
  --config   [./config.yml]
```
