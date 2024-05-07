
# Ethereum Wallet



## Introduction

This Go program is designed to manage Ethereum accounts, including key generation, balance monitoring, and data storage in a MySQL database. It uses the Ethereum JSON-RPC API to check account balances.


## Prerequisites

- Golang 1.22
```bash
$ snap install go --classic
```
- Docker & Docker Swarm https://docs.docker.com/engine/install


## Deploy Ethereum Wallet

#### Clone the repository into a local directory

```bash
$ git clone https://github.com/palmatovic/ethereum-wallet.git
```


#### Navigate to the project directory:

```bash
$ cd ethereum-wallet
```

#### Build image
```bash
$ docker build -t wallet:latest -f image/Dockerfile .
```

#### Review ethereum.yml, replace following values
``` bash
- <YOUR_LOCALNET_IP>
- <SERVER_PORT>
```

#### Deploy
```bash
$ docker stack deploy -c ethereum.yml ethereum
```


## Optional - Dashboard

#### Clone the repository into a local directory

```bash
$ cd Homepage
```

#### Review homepage.yml, replace following value
``` bash
- <UI_PORT>
```

#### Deploy
```bash
$ docker stack deploy -c homepage.yml homepage
```
## Authors

- [@palmatovic](https://www.github.com/palmatovic)

