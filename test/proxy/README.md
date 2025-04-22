General-purpose https reverse proxy.

# Certificates
This tool expects `cert.pem` and `key.pem` to be in the working directory. You may use the example files provided here or put your own files in the working directory.

# Flags
`--addr` - backend server (e.g., https://10.0.0.1)

# Running
## Go Run
`go run main.go --addr https://10.0.0.1`

## Build and Run
```
go build -o proxy main.go
./proxy -addr https://10.0.0.1
```