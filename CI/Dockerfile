FROM golang:1.13
ENV http_proxy=http://proxy-us.intel.com:911
ENV https_proxy=http://proxy-us.intel.com:912
RUN curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/v1.21.0/install.sh | sh -s -- -b $(go env GOPATH)/bin