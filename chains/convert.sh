#!/bin/sh

cat >chains.go <<EOF
package chains

var chainJSON = map[uint64]string{
EOF

curl https://chainid.network/chains.json | jq -c '.[]' | while read foo; do
	echo "	$(echo "$foo" | jq -r .chainId): \`$foo\`," >>chains.go
done

echo "}" >>chains.go
