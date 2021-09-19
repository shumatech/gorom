//  GoRom - Emulator ROM Management Utilities
//  Copyright (C) 2020 Scott Shumate <scott@shumatech.com>
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU General Public License as published by
//  the Free Software Foundation, either version 3 of the License, or
//  (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.
package util

import (
    "testing"
    "gorom/test"
)

func TestHumanizePow2(t *testing.T) {
    nums := []int64{ 0, 1023, 1024, 1025, 1024*1024, 1024*1024-1, 1024*1024*1024, 1024*1024*1024-1}
    strs := []string{ "0 B", "1023 B", "1 KiB", "1.001 KiB", "1 MiB", "1023.999 KiB", "1 GiB", "1 GiB" }

    for i := 0; i < len(nums); i++ {
        s := HumanizePow2(nums[i])
        if s != strs[i] {
            test.Fail(t, "unexpected string " + strs[i] + " " + s)
        }
    }
}

func TestStringBiMap(t *testing.T) {
    sbm := NewStringBiMap()

    if !sbm.Set("k1", "v1") {
        test.Fail(t, "Set method")
    }

    if !sbm.Set("k2", "v2") {
        test.Fail(t, "Set method")
    }

    if sbm.Set("k2", "v3") {
        test.Fail(t, "Set method")
    }

    if sbm.Set("k3", "v2") {
        test.Fail(t, "Set method")
    }

    if v, ok := sbm.Get("k1"); !ok || v != "v1" {
        test.Fail(t, "Get method")
    }

    if v, ok := sbm.Get("k2"); !ok || v != "v2" {
        test.Fail(t, "Get method")
    }

    if _, ok := sbm.Get("k3");  ok {
        test.Fail(t, "Get method")
    }

    if k, ok := sbm.GetValue("v1"); !ok || k != "k1" {
        test.Fail(t, "GetValue method")
    }

    if k, ok := sbm.GetValue("v2"); !ok || k != "k2" {
        test.Fail(t, "GetValue method")
    }

    if _, ok := sbm.GetValue("v3"); ok {
        test.Fail(t, "GetValue method")
    }

    if len(sbm.Keys()) != 2 {
        test.Fail(t, "Keys method")
    }

    if len(sbm.Values()) != 2 {
        test.Fail(t, "Values method")
    }

    if !sbm.Delete("k1", "v1") {
        test.Fail(t, "Delete method")
    }

    if sbm.Delete("k1", "v1") {
        test.Fail(t, "Delete method")
    }

    if !sbm.Delete("k2", "v2") {
        test.Fail(t, "Delete method")
    }

    if sbm.Delete("k2", "v2") {
        test.Fail(t, "Delete method")
    }
}
