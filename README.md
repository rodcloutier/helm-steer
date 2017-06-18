# Helm Steer Plugin

This is a Helm plugin to help deploy and manage multiple charts in a Kubernetes
cluster. Operations are performed according to a plan, a file, that direct the
charts to install, with the appropriate options.

## Usage

Install and upgrade charts in the cluster.

```
$ helm steer plan.yaml
```

## Plan file

`helm steer` use `plan` files to direct the operations. The `plan` file
are akin to a frozen command to Helm with all the flags and arguments.

Each Chart entry contains a dictionnary with the keys being exactly the
same as the helm command flags.

Have a look a the [plan.yaml.tpl](plan.yaml.tpl) for an annoted example
of a plan file.


## Install

```
$ helm plugin install https://github.com/rodcloutier/helm-steer
```

The above will fetch the latest binary release of `helm steer` and install it.


### Developer (From Source) Install

If you would like to handle the build yourself, instead of fetching a binary,
this is how we recommend doing it.

First, set up your environment:

- You need to have [Go](http://golang.org) installed. Make sure to set `$GOPATH`
- If you don't have [Glide](http://glide.sh) installed, this will install it into
  `$GOPATH/bin` for you.

Clone this repo into your `$GOPATH`. You can use `go get -d github.com/rodcloutier/helm-steer`
for that.

```
$ cd $GOPATH/src/github.com/rodcloutier/helm-steer
$ make bootstrap build
$ SKIP_BIN_INSTALL=1 helm plugin install $GOPATH/src/github.com/rodcloutier/helm-steer
```

That last command will skip fetching the binary install and use the one you
built.


## Credits

* [@technosophos](http://github.com/technosophos) from which the repo structure and support files are inspired
