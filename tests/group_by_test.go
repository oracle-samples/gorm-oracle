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
	"testing"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm/utils/tests"
)

func TestGroupBy(t *testing.T) {
	users := []User{{
		Name:     "groupby",
		Age:      10,
		Birthday: tests.Now(),
		Active:   true,
	}, {
		Name:     "groupby",
		Age:      20,
		Birthday: tests.Now(),
	}, {
		Name:     "groupby",
		Age:      30,
		Birthday: tests.Now(),
		Active:   true,
	}, {
		Name:     "groupby1",
		Age:      110,
		Birthday: tests.Now(),
	}, {
		Name:     "groupby1",
		Age:      220,
		Birthday: tests.Now(),
		Active:   true,
	}, {
		Name:     "groupby1",
		Age:      330,
		Birthday: tests.Now(),
		Active:   true,
	}}

	if err := DB.Create(&users).Error; err != nil {
		t.Errorf("errors happened when create: %v", err)
	}

	var name string
	var total int
	if err := DB.Model(&User{}).Select("\"name\", sum(\"age\")").Where("\"name\" = ?", "groupby").Group("name").Row().Scan(&name, &total); err != nil {
		t.Errorf("no error should happen, but got %v", err)
	}

	if name != "groupby" || total != 60 {
		t.Errorf("name should be groupby, but got %v, total should be 60, but got %v", name, total)
	}

	if err := DB.Model(&User{}).Select("\"name\", sum(\"age\")").Where("\"name\" = ?", "groupby").Group("users.name").Row().Scan(&name, &total); err != nil {
		t.Errorf("no error should happen, but got %v", err)
	}

	if name != "groupby" || total != 60 {
		t.Errorf("name should be groupby, but got %v, total should be 60, but got %v", name, total)
	}

	if err := DB.Model(&User{}).Select("\"name\", sum(\"age\") as \"total\"").Where("\"name\" LIKE ?", "groupby%").Group("name").Having("\"name\" = ?", "groupby1").Row().Scan(&name, &total); err != nil {
		t.Errorf("no error should happen, but got %v", err)
	}

	if name != "groupby1" || total != 660 {
		t.Errorf("name should be groupby, but got %v, total should be 660, but got %v", name, total)
	}

	result := struct {
		Name  string
		Total int64
	}{}

	if err := DB.Model(&User{}).Select("\"name\", sum(\"age\") as \"total\"").Where("\"name\" LIKE ?", "groupby%").Group("name").Having("\"name\" = ?", "groupby1").Find(&result).Error; err != nil {
		t.Errorf("no error should happen, but got %v", err)
	}

	if result.Name != "groupby1" || result.Total != 660 {
		t.Errorf("name should be groupby1, total should be 660, but got %+v", result)
	}

	if err := DB.Model(&User{}).Select("\"name\", sum(\"age\") as \"total\"").Where("\"name\" LIKE ?", "groupby%").Group("name").Having("\"name\" = ?", "groupby1").Scan(&result).Error; err != nil {
		t.Errorf("no error should happen, but got %v", err)
	}

	if result.Name != "groupby1" || result.Total != 660 {
		t.Errorf("name should be groupby1, total should be 660, but got %+v", result)
	}

	var active bool
	if err := DB.Model(&User{}).Select("\"name\", \"active\", sum(\"age\")").Where("\"name\" = ? and \"active\" = ?", "groupby", true).Group("name").Group("active").Row().Scan(&name, &active, &total); err != nil {
		t.Errorf("no error should happen, but got %v", err)
	}

	if name != "groupby" || active != true || total != 40 {
		t.Errorf("group by two columns, name %v, age %v, active: %v", name, total, active)
	}
}
