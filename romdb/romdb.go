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
package romdb

import (
    "bytes"
    "errors"
    "fmt"
    "os"
    "path"
    "runtime"
    "time"

    "gorom"
    "gorom/util"
    "gorom/romio"
    "gorom/checksum"
    "gorom/term"

    "github.com/boltdb/bolt"
    "github.com/kelindar/binary"
)

const (
    DbFile = ".gorom.db"
    RomBucket = "rom"
    ChecksumBucket = "checksum"
)

var (
    StopError = errors.New("stopped")
)

type RomDB struct {
    Dir string
    db  *bolt.DB
    skipHeader bool
}

type RomDBInfo struct {
    Checksum checksum.Sha1
    ModTime  time.Time
}

type RomDBEntry struct {
    MachPath string
    RomPath  string
    ModTime  time.Time
    Sum      checksum.Sha1
}

func OpenRomDB(dir string, skipHeader bool) (*RomDB, error) {
    path := path.Join(dir, DbFile)
    db, err := bolt.Open(path, 0644, &bolt.Options{Timeout: 3 * time.Second})
    if err != nil {
        return nil, fmt.Errorf("%s: %s", path, err.Error())
    }

    return &RomDB{ dir, db, skipHeader }, nil
}

func (rdb *RomDB) Close() {
    rdb.db.Close()
}

func (rdb *RomDB) addFiles(machPath string, files []*romio.RomFile, checksums []checksum.Sha1) error {
    return rdb.db.Batch(func(tx *bolt.Tx) error {
        rb, err := tx.CreateBucketIfNotExists([]byte(RomBucket))
        if err != nil {
            return err
        }

        cb, err := tx.CreateBucketIfNotExists([]byte(ChecksumBucket))
        if err != nil {
            return err
        }

        for i, file := range files {
             // Create the database key
            romKey := []byte(machPath + "\x00" + file.Name)

            // If there is already a ROM entry, then delete its checksum lookup
            v := rb.Get(romKey)
            if v != nil {
                // Ignore errors since the lookup function will catch it later
                var oldEntry RomDBEntry
                err = binary.Unmarshal(v, &oldEntry)
                if err == nil {
                    cb.Delete(oldEntry.Sum[:])
                }
            }

            // Add the ROM entry to the database
            entry := RomDBEntry{ machPath, file.Name, file.ModTime, checksums[i] }
            buffer, err := binary.Marshal(&entry)
            if err != nil {
                return err
            }
            err = rb.Put(romKey, buffer)
            if err != nil {
                return err
            }

            // Add the checksum lookup to the database
            err = cb.Put(checksums[i][:], romKey)
            if err != nil {
                return err
            }
        }

        return nil
    })
}

func (rdb *RomDB) deleteAll(machPath string) error {
    return rdb.db.Batch(func(tx *bolt.Tx) error {
        rb := tx.Bucket([]byte(RomBucket))
        if rb == nil {
            return nil
        }

        cb := tx.Bucket([]byte(RomBucket))
        if cb == nil {
            return nil
        }

        delKeys := [][]byte{}
        delSums := [][]byte{}

        machKey := []byte(machPath + "\x00")
        rbc := rb.Cursor()
        for k, v := rbc.Seek(machKey); k != nil && bytes.HasPrefix(k, machKey); k, v = rbc.Next() {
            var entry RomDBEntry
            err := binary.Unmarshal(v, &entry)
            if err == nil {
                delSums = append(delSums, entry.Sum[:])
                delKeys = append(delKeys, k)
            }
        }

        for i, k := range delKeys {
            rb.Delete(k)
            cb.Delete(delSums[i])
        }

        return nil
    })
}

func (rdb *RomDB) deleteChecksum(sum checksum.Sha1) error {
    return rdb.db.Batch(func(tx *bolt.Tx) error {
        cb := tx.Bucket([]byte(RomBucket))
        if cb != nil {
            cb.Delete(sum[:])
        }
        return nil
    })
}

func checksumRom(rr romio.RomReader, rf *romio.RomFile, skipHeader bool) (checksum.Sha1, error) {
    rc, err := rr.Open(rf)
    if err != nil {
        return checksum.Sha1{}, err
    }
    defer rc.Close()

    options := romio.ChecksumNoCrc32
    if skipHeader {
        options = options | romio.ChecksumSkipHeader
    }
    checksums, err := romio.ChecksumRom(rc, options)
    if err != nil {
        return checksum.Sha1{}, err
    }
    return checksums.Sha1, nil
}

func (rdb *RomDB) Dump() {
    rdb.db.View(func(tx *bolt.Tx) error {
        rb := tx.Bucket([]byte(RomBucket))
        if rb != nil {
            c := rb.Cursor();
            for k, v := c.First(); k != nil; k, v = c.Next() {
                var entry RomDBEntry
                err := binary.Unmarshal(v, &entry)
                if err == nil {
                    term.Printf("%s => %s %s %x %v\n", string(k), entry.MachPath, entry.RomPath, entry.Sum, entry.ModTime)
                }
            }
        }
        return nil
    })
}

func (rdb *RomDB) ChecksumArchive(rr romio.RomReader, checksumFunc ChecksumFunc) error {
    files := rr.Files()
    checksums := make([]checksum.Sha1, len(files))
    sumAll := false
    delAll := false
    machPath := path.Base(rr.Path())

    // If any file is out of date or not present, then delete all entries
    // and regenerate all checksums
    err := rdb.db.View(func(tx *bolt.Tx) error {
        rb := tx.Bucket([]byte(RomBucket))
        if rb != nil {
            same := false
            for i, file := range files {
                var entry RomDBEntry

                romKey := []byte(machPath + "\x00" + file.Name)
                val := rb.Get(romKey)
                if val == nil {
                    sumAll = true
                    delAll = true
                    break
                } else {
                    err := binary.Unmarshal(val, &entry)
                    if err == nil {
                        // Compare to milliseconds to avoid rounding issues across filesystems
                        t1 := file.ModTime.Round(time.Millisecond)
                        t2 := entry.ModTime.Round(time.Millisecond)
                        if err == nil && t1.Equal(t2) {
                            same = true
                            checksums[i] = entry.Sum
                        }
                    }
                    if !same {
                        sumAll = true
                        delAll = true
                        break
                    }
                }
            }
        } else {
            sumAll = true
        }
        return nil
    })
    if err != nil {
        return err
    }

    if delAll {
        err = rdb.deleteAll(machPath)
        if err != nil {
            return err
        }
    }

    if sumAll {
        for i, file := range files {
            checksums[i], err = checksumRom(rr, file, rdb.skipHeader)
            if err != nil {
                return err
            }
            if checksumFunc != nil {
                err = checksumFunc(file.Name, checksums[i])
                if err != nil {
                    return err
                }
            }
        }
        // Add the checksums to the database
        err = rdb.addFiles(machPath, files, checksums)
    } else {
        if checksumFunc != nil {
            for i, file := range files {
                err = checksumFunc(file.Name, checksums[i])
                if err != nil {
                    return err
                }
            }
        }
    }

    return nil
}

func (rdb *RomDB) ChecksumDir(rr romio.RomReader, checksumFunc ChecksumFunc) error {
    files := rr.Files()
    machPath := path.Base(rr.Path())
    sumAll := false

    // For each file, if there is no database entry then we need to calc the checksum.
    // If there is a database entry and the modification date is the same, then
    // we can use the database checksum, else recalc the checksum.
    addFiles := []*romio.RomFile{}
    addIndex := []int{}
    err := rdb.db.View(func(tx *bolt.Tx) error {
        rb := tx.Bucket([]byte(RomBucket))
        if rb != nil {
            for i, file := range files {
                same := false
                romKey := []byte(machPath + "\x00" + file.Name)
                val := rb.Get(romKey)
                if val != nil {
                    var entry RomDBEntry
                    err := binary.Unmarshal(val, &entry)
                    if (err == nil) {
                        // Compare to milliseconds to avoid rounding issues across filesystems
                        t1 := file.ModTime.Round(time.Millisecond)
                        t2 := entry.ModTime.Round(time.Millisecond)
                        if t1.Equal(t2) {
                            same = true
                            if checksumFunc != nil {
                                err = checksumFunc(file.Name, entry.Sum)
                                if err != nil {
                                    return err
                                }
                            }
                        }
                    }
                }
                if !same {
                    addFiles = append(addFiles, file)
                    addIndex = append(addIndex, i)
                }
            }
        } else {
            sumAll = true
        }
        return nil
    })
    if err != nil {
        return err
    }

    if sumAll {
        checksums := make([]checksum.Sha1, len(files))
        for i, file := range files {
            checksums[i], err = checksumRom(rr, file, rdb.skipHeader)
            if err != nil {
                return err
            }
            if checksumFunc != nil {
                err = checksumFunc(file.Name, checksums[i])
                if err != nil {
                    return err
                }
            }
        }
        err = rdb.addFiles(machPath, files, checksums)
    } else if len(addFiles) > 0 {
        checksums := make([]checksum.Sha1, len(addFiles))
        for i, file := range addFiles {
            checksums[i], err = checksumRom(rr, file, rdb.skipHeader)
            if err != nil {
                return err
            }
            if checksumFunc != nil {
                err = checksumFunc(file.Name, checksums[i])
                if err != nil {
                    return err
                }
            }
        }
        err = rdb.addFiles(machPath, addFiles, checksums)
    }

    return err
}

type ChecksumFunc func(name string, sum checksum.Sha1) error

func (rdb *RomDB) Checksum(rr romio.RomReader, checksumFunc ChecksumFunc) error {
    if rr.Format() == gorom.FormatDir {
        return rdb.ChecksumDir(rr, checksumFunc)
    } else {
        return rdb.ChecksumArchive(rr, checksumFunc)
    }
}

func (rdb *RomDB) Lookup(checksum checksum.Sha1) (*RomDBEntry, error) {
    var v []byte
    err := rdb.db.View(func(tx *bolt.Tx) error {
        cb := tx.Bucket([]byte(ChecksumBucket))
        if cb == nil {
            return nil
        }

        romKey := cb.Get(checksum[:])
        if romKey == nil {
            return nil
        }

        rb := tx.Bucket([]byte(RomBucket))
        if rb != nil {
            v = rb.Get(romKey)
        }

        v = rb.Get(romKey)
        return nil;
    })
    if err != nil {
        return nil, err
    }
    if v == nil {
        return nil, nil
    }

    var entry RomDBEntry
    err = binary.Unmarshal(v, &entry)
    if err != nil {
        return nil, err
    }

    // Delete the checksum in the bucket if it doesn't match
    if checksum != entry.Sum {
        err = rdb.deleteChecksum(checksum)
        return nil, err
    }

    return &entry, nil
}

type ScanFunc func(machPath string, err error)

type ScanResults struct {
    romKeys util.StringSet
    name string
    err error
}

func isStop(stop chan struct{}) bool {
    if stop == nil {
        return false
    }
    select {
    case <-stop:
        return true
    default:
        return false
    }
}

func (rdb *RomDB) scanChecksum(name string, ch chan ScanResults, stop chan struct{}) {
    if isStop(stop) {
        ch <- ScanResults{ name: name, err: StopError }
        return
    }

    romKeys := util.NewStringSet()
    rr, err := romio.OpenRomReader(path.Join(rdb.Dir, name))
    if err == nil {
        if rr != nil {
            err := rdb.Checksum(rr, func(name string, sum checksum.Sha1) error {
                if isStop(stop) {
                    return StopError;
                }
                return nil
            })
            if err == nil {
                for _, file := range rr.Files() {
                    keyStr := name + "\x00" + file.Name
                    romKeys.Set(keyStr)
                }
            }
            rr.Close()
        }
    }

    ch <- ScanResults{ name: name, romKeys: romKeys, err: err }
}

func scanResults(romKeys util.StringSet, ch chan ScanResults, scanFunc ScanFunc) {
    results := <- ch
    if scanFunc != nil {
        scanFunc(results.name, results.err)
    }
    if results.err == nil {
        for k := range results.romKeys {
            romKeys.Set(k)
        }
    }
}

func (rdb *RomDB) Scan(goLimit int, stop chan struct{}, scanFunc ScanFunc) error {
    ch := make(chan ScanResults, 1)
    romKeys := util.NewStringSet()
    if goLimit <= 0 {
        goLimit = runtime.NumCPU()
    }

    goCount := 0
    err := util.ScanDir(rdb.Dir, true, func(info os.FileInfo) error {
        if goCount == goLimit {
            scanResults(romKeys, ch, scanFunc)
        } else {
            goCount++
        }

        go rdb.scanChecksum(info.Name(), ch, stop)

        if isStop(stop) {
            return StopError;
        }
        return nil
    })
    if err != nil {
        return err
    }

    for ; goCount > 0; goCount-- {
        scanResults(romKeys, ch, scanFunc)
    }

    if isStop(stop) {
        return StopError
    }

    // Remove any database entries not in the set
    return rdb.db.Update(func(tx *bolt.Tx) error {
        rb := tx.Bucket([]byte(RomBucket))
        if rb == nil {
            return nil
        }

        cb := tx.Bucket([]byte(RomBucket))
        if cb == nil {
            return nil
        }

        delKeys := [][]byte{}
        delSums := [][]byte{}

        rbc := rb.Cursor()
        for k, v := rbc.First(); k != nil; k, v = rbc.Next() {
            keyStr := string(k)
            if !romKeys.IsSet(keyStr) {
                delKeys = append(delKeys, k)

                var entry RomDBEntry
                err = binary.Unmarshal(v, &entry)
                if err == nil {
                    delSums = append(delSums, entry.Sum[:])
                }
            }
        }

        for i, k := range delKeys {
            rb.Delete(k)
            cb.Delete(delSums[i])
        }

        return nil
    })
}
