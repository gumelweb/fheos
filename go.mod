module github.com/fhenixprotocol/fheos

go 1.20

require (
	github.com/cockroachdb/pebble v0.0.0-20230928194634-aa077af62593 // indirect
	// needs to match the one in go-ethereum because fuck you that's why
	//github.com/cockroachdb/pebble v1.1.0
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/ethereum/go-ethereum v1.13.3
	github.com/fhenixprotocol/warp-drive/fhe-driver v0.0.0
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.5
)

replace (
	github.com/ethereum/go-ethereum => ./go-ethereum
	github.com/fhenixprotocol/warp-drive/fhe-bridge => ./warp-drive/fhe-bridge
	github.com/fhenixprotocol/warp-drive/fhe-driver => ./warp-drive/fhe-driver
)

require (
	github.com/DataDog/zstd v1.5.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.7.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cockroachdb/errors v1.11.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/consensys/gnark-crypto v0.10.0 // indirect
	github.com/crate-crypto/go-kzg-4844 v0.7.0 // indirect
	github.com/ethereum/c-kzg-4844 v0.4.0 // indirect
	github.com/fhenixprotocol/warp-drive/fhe-bridge v0.0.0-20231205134639-3c799c823a17 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gammazero/deque v0.2.1 // indirect
	github.com/getsentry/sentry-go v0.18.0 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb // indirect
	github.com/holiman/uint256 v1.2.3 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.39.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	github.com/supranational/blst v0.3.11 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/grpc v1.62.1 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

require (
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/cockroachdb/pebble v0.0.0-20230928194634-aa077af62593 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20230807174530-cc333fc44b06 // indirect
	github.com/deckarep/golang-set/v2 v2.1.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/tools v0.13.0 // indirect

)
