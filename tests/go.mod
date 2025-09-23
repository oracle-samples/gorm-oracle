module github.com/oracle-samples/gorm-oracle/tests

go 1.25.1

require gorm.io/gorm v1.31.0

require (
	github.com/godror/godror v0.49.3
	github.com/oracle-samples/gorm-oracle v0.1.0
	github.com/stretchr/testify v1.10.0
	gorm.io/datatypes v1.2.6
	github.com/google/uuid v1.6.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/VictoriaMetrics/easyproto v0.1.4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/godror/knownpb v0.3.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/exp v0.0.0-20250911091902-df9299821621 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/mysql v1.5.6 // indirect
)

replace github.com/oracle-samples/gorm-oracle => ../