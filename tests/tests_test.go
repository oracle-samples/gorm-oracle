/*
** Copyright (c) 2025 Oracle and/or its affiliates.
**
** The Universal Permissive License (UPL), Version 1.0
**
** Subject to the condition set forth below, permission is hereby granted to any
** person obtaining a copy of this software, associated documentation and/or data
** (collectively the "Software"), free of charge and under any and all copyright
** rights in the Software, and any and all patent rights owned or freely
** licensable by each licensor hereunder covering either (i) the unmodified
** Software as contributed to or provided by such licensor, or (ii) the Larger
** Works (as defined below), to deal in both
**
** (a) the Software, and
** (b) any piece of software and/or hardware listed in the lrgrwrks.txt file if
** one is included with the Software (each a "Larger Work" to which the Software
** is contributed by such licensors),
**
** without restriction, including without limitation the rights to copy, create
** derivative works of, display, perform, and distribute the Software and make,
** use, sell, offer for sale, import, export, have made, and have sold the
** Software and the Larger Work(s), and to sublicense the foregoing rights on
** either these or other terms.
**
** This license is subject to the following condition:
** The above copyright notice and either this complete permission notice or at
** a minimum a reference to the UPL must be included in all copies or
** substantial portions of the Software.
**
** THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
** IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
** FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
** AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
** LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
** OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
** SOFTWARE.
 */

package tests

import (
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/oracle-samples/gorm-oracle/oracle"
	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

var newLogger = logger.New(
	log.New(os.Stdout, "\r\n", log.LstdFlags),
	logger.Config{
		SlowThreshold:             time.Second,
		LogLevel:                  logger.Info,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      false,
		Colorful:                  true,
	},
)

var oracleDSN = `user="system" password="manager"
	connectString="phoenix793630.dev3sub1phx.databasede3phx.oraclevcn.com:1661/cdb1_pdb1.regress.rdbms.dev.us.oracle.com"
	libDir="/home/hkhasnis/projects/pkgs/23IC"`

func init() {
	var err error
	if DB, err = OpenTestConnection(&gorm.Config{Logger: newLogger}); err != nil {
		log.Printf("failed to connect database, got error %v", err)
		os.Exit(1)
	} else {
		sqlDB, err := DB.DB()
		if err != nil {
			log.Printf("failed to connect database, got error %v", err)
			os.Exit(1)
		}

		err = sqlDB.Ping()
		if err != nil {
			log.Printf("failed to ping sqlDB, got error %v", err)
			os.Exit(1)
		}

		// Print GORM environment information after successful connection
		GormInfo()

		RunMigrations()
	}
}

func OpenTestConnection(cfg *gorm.Config) (db *gorm.DB, err error) {
	dbDSN := os.Getenv("GORM_DSN")
	log.Println("testing oracle...")
	if dbDSN == "" {
		dbDSN = oracleDSN
	}
	db, err = gorm.Open(oracle.Open(dbDSN), cfg)

	if err != nil {
		return
	}

	if debug := os.Getenv("DEBUG"); debug == "true" {
		db.Logger = db.Logger.LogMode(logger.Info)
	} else if debug == "false" {
		db.Logger = db.Logger.LogMode(logger.Silent)
	}

	return
}

func RunMigrations() {
	var err error
	allModels := []interface{}{&User{}, &Account{}, &Pet{}, &Company{}, &Toy{}, &Language{}, &Coupon{}, &CouponProduct{}, &Order{}, &Parent{}, &Child{}, &Tools{}}
	seed := int64(time.Now().UnixNano())
	rand.New(rand.NewSource(seed))
	rand.Shuffle(len(allModels), func(i, j int) { allModels[i], allModels[j] = allModels[j], allModels[i] })

	DB.Migrator().DropTable("user_friends", "user_speaks")

	// DropTable returns ORA-00942 if table doesn't exist
	for _, model := range allModels {
		if err = DB.Debug().Migrator().DropTable(model); err != nil {
			if !strings.Contains(err.Error(), "ORA-00942") {
				log.Printf("Failed to drop table, got error %v\n", err)
				os.Exit(1)
			}
		}
	}

	if err = DB.AutoMigrate(allModels...); err != nil {
		log.Printf("Failed to auto migrate, but got error %v\n", err)
		os.Exit(1)
	}

	for _, m := range allModels {
		if !DB.Migrator().HasTable(m) {
			log.Printf("Failed to create table for %#v\n", m)
			os.Exit(1)
		}
	}
}
