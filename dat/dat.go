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
package dat

import (
    "bufio"
    "encoding/xml"
    "fmt"
    "io"
    "os"
    "path"
    "strings"

    "compress/gzip"

    "gorom/romdb"
    "gorom/romio"
    "gorom/checksum"
)

///////////////////////////////////////////////////////////////////////////////
// DAT file XML types
///////////////////////////////////////////////////////////////////////////////
type Header struct {
    Name            string         `xml:"name"`
    Description     string         `xml:"description"`
    Version         string         `xml:"version"`
    Author          string         `xml:"author"`
}

type Machine struct {
    Name            string         `xml:"name,attr"`
    Description     string         `xml:"description"`
    Year            string         `xml:"year"`
    Manufacturer    string         `xml:"manufacturer"`
    Category        string         `xml:"category"`
    Roms            []*Rom         `xml:"rom"`
    Path            string
    Format          int
}

type Rom struct {
    Name            string         `xml:"name,attr"`
    Size            int64          `xml:"size,attr"`
    Crc             checksum.Crc32 `xml:"crc,attr"`
    Sha1            checksum.Sha1  `xml:"sha1,attr"`
    Status          int
}

// Status constants
const (
    RomUnknown = iota
    RomOk
    RomCorrupt
    RomMissing
    RomBadName
)

///////////////////////////////////////////////////////////////////////////////
// DAT path conversion
///////////////////////////////////////////////////////////////////////////////
func ToDatPath(path string) string {
    return strings.ReplaceAll(path, "/", "\\")
}

func FromDatPath(path string) string {
    return strings.ReplaceAll(path, "\\", "/")
}

///////////////////////////////////////////////////////////////////////////////
// Checksum map
///////////////////////////////////////////////////////////////////////////////

type ChecksumMap struct {
    toChecksum map[string]checksum.Sha1
    toName map[checksum.Sha1]string
}

func NewChecksumMap() *ChecksumMap {
    return &ChecksumMap {
        toChecksum: map[string]checksum.Sha1{},
        toName: map[checksum.Sha1]string{} }
}

func (cm *ChecksumMap) Add(name string, sum checksum.Sha1) {
    cm.toChecksum[name] = sum
    cm.toName[sum] = name
}

func (cm *ChecksumMap) Delete(name string, sum checksum.Sha1) {
    delete(cm.toChecksum, name)
    delete(cm.toName, sum)
}

func (cm *ChecksumMap) ToChecksum(name string) (sum checksum.Sha1, ok bool) {
    sum, ok = cm.toChecksum[name]
    return
}

func (cm *ChecksumMap) ToName(sum checksum.Sha1) (name string, ok bool) {
    name, ok = cm.toName[sum]
    return
}

func (cm *ChecksumMap) ForEach(callback func(name string, sum checksum.Sha1)) {
    for name, sum := range cm.toChecksum {
        callback(name, sum)
    }
}

///////////////////////////////////////////////////////////////////////////////
// ValidateChecksums - Validate the presence, SHA1 checksum, and name for each
// ROM in a machine. ROMs are contained in either an archive file or a directory
// that has the same name as the machine.  Validation results are set in the
// Status field for each ROM.
//
// ROMs with bad names are determined by checksum and are added to the
// badNames map (if non-nil). Extraneous ROMs not present in the machine are
// added to the extras slice (if non-nil).
//
// For each ROM found, the checksumFunc is called with the generated checksum.
// The function returns true if the machine dir/zip was found and scanned
// successfully and false if not.
///////////////////////////////////////////////////////////////////////////////

func ValidateChecksums(machine *Machine, rdb *romdb.RomDB, badNames map[string]string,
                       extras *[]string, checksumFunc romdb.ChecksumFunc) (bool, error) {
    romMap := NewChecksumMap();
    machName := machine.Name

    rr, err := romio.OpenRomReaderByName(machName)
    if rr == nil || err != nil {
        return false, err
    }
    defer rr.Close()

    machine.Format = rr.Format()
    machine.Path = rr.Path()

    err = rdb.Checksum(rr, func(name string, checksum checksum.Sha1) error {
        romMap.Add(name, checksum)
        if checksumFunc != nil {
            return checksumFunc(name, checksum)
        }
        return nil;
    })
    if err != nil {
        return false, err
    }

    // Set the status field for each ROM based on our results
    for index, rom := range machine.Roms {
        if checksum, ok := romMap.ToChecksum(rom.Name); ok {
            if checksum == rom.Sha1 {
                machine.Roms[index].Status = RomOk
            } else {
                machine.Roms[index].Status = RomCorrupt
            }
            romMap.Delete(rom.Name, checksum)
        } else {
            if name, ok := romMap.ToName(rom.Sha1); ok {
                machine.Roms[index].Status = RomBadName
                if badNames != nil {
                    badNames[rom.Name] = name
                }
                romMap.Delete(name, rom.Sha1)
            } else {
                machine.Roms[index].Status = RomMissing
            }
        }
    }

    // Any ROMs left in the map are extraneous
    if extras != nil {
        romMap.ForEach(func(name string, checksum checksum.Sha1) {
            *extras = append(*extras, name)
        })
    }

    return true, nil
}

///////////////////////////////////////////////////////////////////////////////
// ValidateSizes - Validate the presence, size, and name for each ROM in a
// machine. The checksum is NOT validated which makes this function much faster
// than ValidateChecksum but not authoritative.  Validation results are set in
// the Status field for each ROM.
//
// ROMs with bad names cannot be found since the checksum is not calculated so
// they will end up as extraneous ROMs. Also, corrupt ROMs with a matching
// size will not be detected.  Extraneous ROMs are added to the extras slice
// (if non-nil).
//
// For each ROM found, the sizeFunc is called with the file size. The function
// returns true if the machine dir/zip was found and scanned successfully and
// false if not.
///////////////////////////////////////////////////////////////////////////////

type SizeFunc func(name string, size int64)

func ValidateSizes(machine *Machine, extras *[]string, sizeFunc SizeFunc) (bool, error) {
    romMap := make(map[string]int)
    for index, rom := range machine.Roms {
        romMap[rom.Name] = index
    }

    machName := machine.Name

    rr, err := romio.OpenRomReaderByName(machName)
    if rr == nil || err != nil {
        return false, err
    }
    defer rr.Close()

    machine.Format = rr.Format()
    machine.Path = rr.Path()

    for _, file := range rr.Files() {
        index, ok := romMap[file.Name]
        if ok {
            if file.Size == machine.Roms[index].Size {
                machine.Roms[index].Status = RomOk
            } else {
                machine.Roms[index].Status = RomCorrupt
            }
            if sizeFunc != nil {
                sizeFunc(file.Name, file.Size)
            }
        } else {
            *extras = append(*extras, file.Name)
        }
    }

    for index, rom := range machine.Roms {
        if (rom.Status == RomUnknown) {
            machine.Roms[index].Status = RomMissing
        }
    }

    return true, nil
}

///////////////////////////////////////////////////////////////////////////////
// ParseDatFile - Parses a ROM dat file or a dir2dat file and executes a
// callback for the header and also a callback for each machine found.  If any
// callback returns an error, then parsing stops and the error is propogated
// up.
///////////////////////////////////////////////////////////////////////////////

type HeaderFunc func(header *Header) error
type MachFunc func(machine *Machine) error

func normalizeRomNames(machine *Machine) {
    machName := machine.Name
    for i, rom := range machine.Roms {
        romName := FromDatPath(rom.Name)

        // Remove the machine name if it is in front of the ROM name
        if strings.HasPrefix(romName, machName + "/") {
            romName = rom.Name[len(machName) + 1:]
        }
        machine.Roms[i].Name = romName
    }
}

func ParseDatFile(datFile string, machFilter []string, headerFunc HeaderFunc, machFunc MachFunc) error {
    df, err := os.Open(datFile)
    if err != nil {
        return err
    }
    defer df.Close()

    var buffer io.Reader
    if path.Ext(datFile) == ".gz" {
        gz, err := gzip.NewReader(df)
        if err != nil {
            return err
        }
        defer gz.Close()
        buffer = bufio.NewReader(gz)
    } else {
        buffer = bufio.NewReader(df)
    }

    decoder := xml.NewDecoder(buffer)

    machMap := make(map[string]bool)
    for _, arg := range machFilter {
        machName := romio.MachName(arg)
        machMap[machName] = true
    }

    machCount := len(machFilter)
    datafile := false
    for {
        tok, _ := decoder.Token()
        if tok == nil {
            break
        }

        se, ok := tok.(xml.StartElement)
        if ok {
            if se.Name.Local == "datafile" {
                datafile = true
            } else if datafile {
                if se.Name.Local == "header" {
                    var header Header
                    decoder.DecodeElement(&header, &se)
                    if headerFunc != nil {
                        err = headerFunc(&header)
                        if err != nil {
                            return nil
                        }
                    }
                } else if se.Name.Local == "machine" || se.Name.Local == "game" {
                    var machine Machine
                    if (machCount > 0) {
                        for _, attr := range se.Attr {
                            if attr.Name.Local == "name" {
                                if _, ok := machMap[attr.Value]; ok {
                                    machCount--
                                    decoder.DecodeElement(&machine, &se)
                                    normalizeRomNames(&machine)
                                    err = machFunc(&machine)
                                    if err != nil {
                                        return err
                                    }
                                }
                                break
                            }
                        }
                        if (machCount == 0) {
                            break
                        }
                    } else {
                        decoder.DecodeElement(&machine, &se)
                        normalizeRomNames(&machine)
                        err = machFunc(&machine)
                        if err != nil {
                            return err
                        }
                    }
                }
            }
        }
    }

    if !datafile {
        return fmt.Errorf("invalid dat file format")
    }

    return nil
}
