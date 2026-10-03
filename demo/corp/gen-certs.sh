#!/bin/sh
# Generates the demo PKI into /certs (once):
#   corp-ca      company root CA, pushed to devices by MDM; the gate signs with it
#   internet-ca  stands in for public CAs; signs the fake internet's server cert
set -eu
apk add --no-cache openssl >/dev/null
cd /certs
[ -f done ] && exit 0

ca() {
  openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 30 \
    -keyout "$1.key" -out "$1.crt" -subj "/CN=$2" \
    -addext "basicConstraints=critical,CA:TRUE" -addext "keyUsage=critical,keyCertSign,cRLSign" 2>/dev/null
}
ca corp-ca "Acme Bank Corporate Root CA"
ca internet-ca "Simulated Public CA"

openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
  -keyout internet.key -out internet.csr -subj "/CN=internet" 2>/dev/null
cat > internet.ext <<EXT
subjectAltName=DNS:api.anthropic.com,DNS:llm.sketchy-vps.example,DNS:news.example
basicConstraints=CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=serverAuth
authorityKeyIdentifier=keyid
EXT
openssl x509 -req -in internet.csr -CA internet-ca.crt -CAkey internet-ca.key -CAcreateserial \
  -days 30 -extfile internet.ext -out internet.crt 2>/dev/null

chmod 644 ./*
touch done
echo "demo PKI ready"
