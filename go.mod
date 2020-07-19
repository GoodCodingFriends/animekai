module github.com/GoodCodingFriends/animekai

go 1.14

require (
	github.com/Yamashou/gqlgenc v0.0.0-20200714143123-f3db1bb60aa0
	github.com/agnivade/levenshtein v1.1.0 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/golangci/golangci-lint v1.27.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/k0kubun/pp v3.0.1+incompatible
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/machinebox/graphql v0.2.2
	github.com/matryer/is v1.3.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1
	github.com/morikuni/failure v0.12.1
	github.com/nametake/protoc-gen-gohttp v1.2.0
	github.com/rakyll/statik v0.1.7
	github.com/rs/cors v1.7.0
	github.com/slack-go/slack v0.6.4
	github.com/yhat/scrape v0.0.0-20161128144610-24b7890b0945
	go.uber.org/zap v1.15.0
	golang.org/x/net v0.0.0-20200625001655-4c5254603344
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/tools v0.0.0-20200717024301-6ddee64345a6 // indirect
	google.golang.org/genproto v0.0.0-20200715011427-11fb19a81f2c // indirect
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.25.0
)

replace github.com/nametake/protoc-gen-gohttp => github.com/ktr0731/protoc-gen-gohttp v1.1.1-0.20200711155709-7f5687c95bf3
