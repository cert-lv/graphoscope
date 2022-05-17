## Hardware requirements

As the service doesn't actively store all the results from the data sources, there is no need for a large disk space. An average server with `4GB RAM` and `50GB HDD` will be good enough to start.


## Database

Continue as **root**. `MongoDB` is being used as a local database. Install the latest version according the official tutorial: https://docs.mongodb.com/manual/administration/install-on-linux/.

Start and enable a database service:
```sh
systemctl start mongod
systemctl enable mongod
```

MongoDB setup:

1. Run a database client:
```sh
mongo
```
Make sure its version is **4.2** or later.

2. Create a new database:
```
use graphoscope
```
3. Add a user with minimal needed permissions:
```sh
db.createUser(
  {
    user: "graphoscope",
    pwd: passwordPrompt(),
    roles: [
       { role: "readWrite", db: "graphoscope" }
    ]
  }
)
```
... enter a password when asked and exit the MongoDB shell.

4. Edit `/etc/mongod.conf` to enable authorization:
```yaml
security:
  authorization: enabled
```
5. Restart the service to apply changes:
```sh
systemctl restart mongod
```


## Get the source code

Create directories and copy the source in there:
```sh
mkdir -p /opt/go/src/github.com/cert-lv
cd /opt/go/src/github.com/cert-lv
git clone https://github.com/cert-lv/graphoscope
mkdir -p build/plugins
```


## Scripted building

`Makefile` and `Docker` are used to test, build and deploy Graphoscope on a remote server. Make sure to install the latest version.

```sh
cd /opt/go/src/github.com/cert-lv/graphoscope/
cp Makefile.example Makefile
```
and edit `Makefile`s according to your needs: set a `REMOTE` variable to your remote user and host, replace `docker` command with `podman` in case it's being used in your system.


## Development host setup

> :warning: To simplify things here we use the same database. Recommendation is to use a different MongoDB instance. A local one, for example.

Configure a Graphoscope service:
```sh
cd /opt/go/src/github.com/cert-lv/graphoscope/
cp sources/demo.yaml.example sources/demo.yaml
cp files/groups.json.example files/groups.json
cp files/formats.yaml.example files/formats.yaml
cp graphoscope.yaml.example graphoscope.yaml
cp Dockerfile.example Dockerfile
```
Edit `graphoscope.yaml`:

- Set database's `user/password` from the previous setup
- Enter a unique `authenticationKey`, `encryptionKey ` in a `sessions` section
- Set `certFile` and `keyFile` to `certs/graphoscope.crt` and `certs/graphoscope.key`

Install the latest official version of `Golang` and run:
```sh
export GOPATH=/opt/go
apt install gcc make
go get
make plugins-local
go run *.go
```

Open in a browser: `https://server:443`, where **server** is your host IP.


## Production server setup

Dev. host can be used to deploy the necessary files on a prod. server, local installation also is possible. On the prod. server install a musl, C standard library.

On DEB based systems:
```sh
apt install musl-dev
ln -s /usr/lib/x86_64-linux-musl/libc.so /lib/libc.musl-x86_64.so.1
```
<!-- On RPM based systems:
```sh
dnf install musl-devel
ln -s /lib/ld-musl-x86_64.so.1 /lib/libc.musl-x86_64.so.1
``` -->

To deploy from a dev. host:
```sh
cd $GOPATH/src/github.com/cert-lv/graphoscope
make compile
make install-remote
```
With a local installation copy a source directory to the remote host, switch to it and run:
```sh
make install
```
Edit remote `/etc/graphoscope/graphoscope.yaml` according to your needs and paths:

- database's `url: mongodb://localhost:27017`, `user/password` from the previous steps
- unique `authenticationKey`, `encryptionKey ` in a `sessions` section. The last one must be exactly **18** characters long
- leave `environment: dev` at the moment


Start the service:
```sh
systemctl start graphoscope
systemctl enable graphoscope
```

Now there is an HTTPS service running on port `TCP 443`. If there are no errors - replace default `graphoscope.crt` and `graphoscope.key` with your own HTTPS cert & key and restart a Graphoscope service:
```sh
systemctl restart graphoscope
```

It is useful from time to time to remove all dangling docker images to free disk space:
```sh
docker image prune
docker volume prune
```


# Updating

Run from a source directory to update a local installation:
```sh
make compile
make update
systemctl start graphoscope
```
or to update a remote server:
```sh
make compile
make update-remote
ssh root@<server-ip> systemctl start graphoscope
```
... where `<server-ip>` is a remote host.


## Postinstallation setup

Sign up to the Web GUI, press top-right **Options** icon and follow the documentation section `Administration` to set administrators and connect your own data sources. After that in `/etc/graphoscope/graphoscope.yaml` you can set `environment: prod` and restart the Graphoscope service.
