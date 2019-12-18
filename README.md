# helferlein

![Helferlein (Little Helper)](./media/helferlein.jpg)

> A small but helpful tool to set up a continuously-running git repository watcher to build CI/CD pipelines in seconds

## features

- configurable poll-rate to keep in sync with upstream changes
- support for concurrent watching and working with as many repositories as your system can handle

## use cases

If you ever wanted to continuously watch git repository changes without a webhook setup or a complete CI/CD solution, this will be for you. I've created helferlein to simplify deployment workflows to my personal infrastructure, basically a self-built and self-hosted CI solution without all the additional bells and whistles.

## installation

Since I went ahead and cross-compiled helferlein for all systems imaginable, you can go ahead and download the OS- and architecture-specific binary from the [release page](https://github.com/BrunoScheufler/helferlein/releases). You can also just clone and build helferlein for yourself, the only important part is to build the `cmd` package (not to be confused with go's internal cmd package!), since this will boot everything up for you.

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
      # React to pushes to the master branch
      master:
        # Run the following commands in order
        steps:
          - echo "Hooray, we've got changes 🎉"
          - bash ./my-script.sh # commands are run in the cloned repository
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
