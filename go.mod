module github.com/begetan/lnb

go 1.14

require (
	github.com/begetan/lnb/client v0.0.0-00010101000000-000000000000
	github.com/btcsuite/btcutil v1.0.2
	github.com/btcsuite/btcwallet/walletdb v1.3.3 // indirect
	github.com/btcsuite/btcwallet/wtxmgr v1.2.0 // indirect
	github.com/coreos/bbolt v1.3.5 // indirect
	github.com/coreos/etcd v3.3.25+incompatible // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/go-errors/errors v1.1.1 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.14.7 // indirect
	github.com/jonboulle/clockwork v0.2.0 // indirect
	github.com/juju/loggo v0.0.0-20200526014432-9ce3a2e09b5e // indirect
	github.com/kkdai/bstream v1.0.0 // indirect
	github.com/lightninglabs/protobuf-hex-display v1.3.3-0.20191212020323-b444784ce75d
	github.com/lightningnetwork/lnd v0.11.0-beta
	github.com/lightningnetwork/lnd/queue v1.0.4 // indirect
	github.com/ltcsuite/ltcd v0.20.1-beta // indirect
	github.com/miekg/dns v1.1.31 // indirect
	github.com/prometheus/common v0.13.0 // indirect
	github.com/shopspring/decimal v1.2.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20200427203606-3cfed13b9966 // indirect
	github.com/urfave/cli/v2 v2.2.0
	go.etcd.io/bbolt v1.3.5 // indirect
	go.uber.org/zap v1.15.0 // indirect
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a // indirect
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6 // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	google.golang.org/genproto v0.0.0-20200827165113-ac2560b5e952 // indirect
	google.golang.org/grpc v1.31.1
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/macaroon-bakery.v2 v2.2.0 // indirect
	gopkg.in/macaroon.v2 v2.1.0 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace github.com/begetan/lnb/client => ./client

replace github.com/coreos/bbolt v1.3.5 => go.etcd.io/bbolt v1.3.5
replace go.etcd.io/bbolt v1.3.5 => github.com/coreos/bbolt v1.3.5

