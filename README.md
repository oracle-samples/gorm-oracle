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

## Documentation

### OnUpdate Foreign Key Constraint

Since Oracle doesn’t support `ON UPDATE` in foreign keys, the driver simulates it using **triggers**.

When a field has a constraint tagged with `OnUpdate`, the driver:

1. Skips generating the unsupported `ON UPDATE` clause in the foreign key definition.
2. Creates a trigger on the parent table that automatically cascades updates to the child table(s) whenever the referenced column is changed.

The `OnUpdate` tag accepts the following values (case-insensitive): `CASCADE`, `SET NULL`, and `SET DEFAULT`.

Take the following struct for an example:

```go
type Profile struct {
  ID    uint
  Name  string
  Refer uint
}

type Member struct {
  ID        uint
  Name      string
  ProfileID uint
  Profile   Profile `gorm:"Constraint:OnUpdate:CASCADE"`
}
```

Trigger SQL created by the driver when migrating:

```sql
CREATE OR REPLACE TRIGGER "fk_trigger_profiles_id_members_profile_id"
AFTER UPDATE OF "id" ON "profiles"
FOR EACH ROW
BEGIN
  UPDATE "members"
  SET "profile_id" = :NEW."id"
  WHERE "profile_id" = :OLD."id";
END;
```

## Contributing

This project welcomes contributions from the community. Before submitting a pull request, please [review our contribution guide](./CONTRIBUTING.md)

## Security

Please consult the [security guide](./SECURITY.md) for our responsible security vulnerability disclosure process

## License

Copyright (c) 2025 Oracle and/or its affiliates. Released under the Universal Permissive License v1.0 as shown at <https://oss.oracle.com/licenses/upl/>.
