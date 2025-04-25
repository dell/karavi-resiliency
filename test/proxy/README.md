General-purpose https reverse proxy.

# Certificates
This tool generates `cert.pem` and `key.pem` in the working directory. Do not commit these files to source control.

# Flags
`--addr` - backend server (e.g., https://10.0.0.1)

# Running
## Go Run
`go run main.go --addr https://10.0.0.1`

## Build and Run
```
go build -o proxy main.go
./proxy --addr https://10.0.0.1
```

The proxy will be running on port 8080 on the machine.

# Driver Configuration
The endpoint in the driver secret (e.g., vxflexos-config) must be the address of the machine running the reverse proxy on port 8080. If you are running the proxy on `10.2.2.2`, the endpoint in the driver secret should be `https://10.2.2.2:8080`.
