# lnb
lightningnetwork/lnd compatible tool for channels balances

## Usage 

```bash
./lnb

 lnb
NAME:
   lnb - lighting channel balancer for lnd

USAGE:
   lnb [global options] command [command options] [arguments...]

COMMANDS:
   get, g   balance, status
   list, l  channels, contracts
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
  -- <the same as lncli>
```

### List of channels' balances
```bash
# From the LND home directory
lnb list channels

# From another directory, e.g. clonned repository
./lnb --lnddir /home/bitcoin/.lnd/ list channels
```
![list channels](https://user-images.githubusercontent.com/17225934/91498171-971ed800-e8bf-11ea-9efe-f563a8049de4.png)

### List of forwarded contracts (HTLCs)
```bash
# From the LND home directory
lnb list contracts

# From another directory
./lnb --lnddir /home/bitcoin/.lnd/ list contracts
```
![list contracts](https://user-images.githubusercontent.com/17225934/91498829-c41fba80-e8c0-11ea-831d-2bf269c5fde6.png)

### Install
First you need Go compiler

```bash
git clone git@github.com:begetan/lnb.git
go build
./lnb get status
```
Status should return the same information as `lncli getinfo`

### About
This tool is aimed for better analysis of lightning network nodes runs on [LND daemon](https://github.com/lightningnetwork/lnd).
You may use the code as an example of grpc LND client for your application

This is a very early stage. Some updates could be implemented futher
* To make a self-payment loop for re-balancing of channels
* Bash auto-completion
* CSV export for external analysys
* CSV export for an external analytic
* You may know