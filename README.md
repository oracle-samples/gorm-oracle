# GORM Driver for Oracle

The GORM Driver for Oracle provides support for Oracle databases, enabling full compatibility with GORM's ORM capabilities. It is built on top of the [Go DRiver for ORacle (Godror)](https://github.com/godror/godror) and supports key features such as auto migrations, associations, transactions, and advanced querying.

## Prerequisite

### Install Instant Client

To use ODPI-C with Godror, you’ll need to install the Oracle Instant Client on your system.

Follow the steps on [this page](https://odpi-c.readthedocs.io/en/latest/user_guide/installation.html) complete the installation.

After that, use a logfmt-encoded parameter list to specify the instanct client directory in the `dataSourceName` when you connect to the database. For example:

```go
dsn := `user="scott" password="tiger" 
        connectString="[host]:[port]/cdb1_pdb1.regress.rdbms.dev.us.oracle.com"
        libDir="/Path/to/your/instantclient_23_8"`
```

## Getting Started

```go main.go
package main

import (
        "github.com/oracle-samples/gorm-oracle/oracle"
        "gorm.io/gorm"
)

func main() {
        dsn := `user="scott" password="tiger"
                connectString="[host]:[port]/cdb1_pdb1.regress.rdbms.dev.us.oracle.com"
                libDir="/Path/to/your/instantclient_23_8"`
        db, err := gorm.Open(oracle.Open(dsn), &gorm.Config{})
}
```

## Contributing

This project welcomes contributions from the community. Before submitting a pull request, please [review our contribution guide](./CONTRIBUTING.md)

## Security

Please consult the [security guide](./SECURITY.md) for our responsible security vulnerability disclosure process

## License

Copyright (c) 2025 Oracle and/or its affiliates. Released under the Universal Permissive License v1.0 as shown at <https://oss.oracle.com/licenses/upl/>.
