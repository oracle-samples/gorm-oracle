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
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/oracle/gorm-oracle/oracle"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormInfo prints Oracle database and client information
// Useful for debugging and environment verification during tests
func GormInfo() {
	now := time.Now().Format(time.RFC1123Z)
	fmt.Printf("=== GORM Environment Information ===\n")
	fmt.Printf("Run at                        : %s\n", now)
	fmt.Printf("Go version                    : %s (%s/%s)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	user := os.Getenv("GORM_ORACLEDB_USER")
	password := os.Getenv("GORM_ORACLEDB_PASSWORD")
	connectString := os.Getenv("GORM_ORACLEDB_CONNECTSTRING")
	libDir := os.Getenv("GORM_ORACLEDB_LIBDIR")

	// Check if all required env vars are set
	if user == "" || password == "" || connectString == "" || libDir == "" {
		fmt.Printf("Skipping Oracle connection info - missing required environment variables\n")
		fmt.Printf("==========================================\n\n")
		return
	}

	dsn := fmt.Sprintf(`user="%s" password="%s" connectString="%s" libDir="%s"`, user, password, connectString, libDir)
	db, err := gorm.Open(oracle.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Printf("Failed to connect for info display: %v", err)
		fmt.Printf("==========================================\n\n")
		return
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// Get ORACLE DATABASE VERSION
	var banner, bannerFull, bannerLegacy string
	var conID int
	err = sqlDB.QueryRow(`SELECT BANNER, BANNER_FULL, BANNER_LEGACY, CON_ID FROM v$version WHERE banner LIKE 'Oracle Database%'`).Scan(&banner, &bannerFull, &bannerLegacy, &conID)
	if err != nil {
		fmt.Printf("Failed to get Oracle DB version: %v\n", err)
	} else {
		fmt.Printf("Oracle Database version       : %s\n", bannerFull)
	}

	// Get CLIENT_DRIVER
	var clientDriver string
	err = sqlDB.QueryRow(`
        SELECT UNIQUE CLIENT_DRIVER
        FROM V$SESSION_CONNECT_INFO
        WHERE SID = SYS_CONTEXT('USERENV', 'SID')
    `).Scan(&clientDriver)
	if err != nil {
		fmt.Printf("Failed to get CLIENT_DRIVER: %v\n", err)
	} else {
		fmt.Printf("Client Driver                 : %s\n", clientDriver)
	}

	// Get client version info from database
	var clientVersion2 string
	err = sqlDB.QueryRow(`
        SELECT UNIQUE CLIENT_VERSION
        FROM V$SESSION_CONNECT_INFO
        WHERE SID = SYS_CONTEXT('USERENV', 'SID')
        AND CLIENT_VERSION IS NOT NULL
    `).Scan(&clientVersion2)
	if err == nil && clientVersion2 != "" {
		fmt.Printf("Oracle Client library version : %s\n", clientVersion2)
	}

	fmt.Printf("==========================================\n\n")
}
