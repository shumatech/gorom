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
package main

import (
    "fmt"
    "log"
    "os"
    "path/filepath"

    "gorom"
    "gorom/term"

    "github.com/jessevdk/go-flags"
)

///////////////////////////////////////////////////////////////////////////////
// Main
///////////////////////////////////////////////////////////////////////////////

type Options struct {
    Operations struct {
        ChkRom      string    `short:"c" long:"chkrom"  description:"Check validity of ROMs in DATFILE" value-name:"DATFILE"`
        FixRom      string    `short:"f" long:"fixrom"  description:"Fix the ROMs in DATFILE" value-name:"DATFILE"`
        ChkTor      string    `short:"t" long:"chktor"  description:"Check the validity of the files in TORRENT" value-name:"TORRENT"`
        LsTor       string    `short:"l" long:"lstor"   description:"List the contents of TORRENT" value-name:"TORRENT"`
        TorZip      bool      `short:"z" long:"torzip"  description:"Convert specified Zips into TorrentZip format"`
        Dir2Dat     bool      `short:"d" long:"dir2dat" description:"Create a DAT file for the current directory"`
        FltDat      string    `short:"F" long:"fltdat"  description:"Filter DATFILE fields with regular expressions" value-name:"DATFILE"`
        FuzzyMv     bool      `short:"m" long:"fuzzymv" description:"Rename files in one directory to the closest fuzzy\nmatch in another directory"`
        GoRomDB     bool      `short:"G" long:"goromdb" description:"Perform operations on the .gorom.db database"`
    } `group:"Operations"`

    App struct {
        NoProgress  bool      `short:"p" long:"no-progress" description:"Do not show progress output"`
        NoColor     bool      `short:"C" long:"no-color" description:"Suppress color output"`
        NoGo        bool      `short:"g" long:"no-go" description:"Do not use parallel go routines"`
        NoOk        bool      `short:"o" long:"no-ok" description:"Do not display OK roms"`
        NoHeader    bool      `short:"H" long:"no-header" description:"Do not display header description"`
        NoExtra     bool      `short:"e" long:"no-extra" description:"Do not show extra files"`
        Verbose     bool      `short:"v" long:"verbose" description:"Show verbose output"`
    } `group:"Application Options"`

    ChkRom struct {
        NoRom       bool      `short:"r" long:"no-rom" description:"Do not display individual roms"`
        NoStats     bool      `short:"T" long:"no-stats" description:"Do not display statistics"`
        JsonOut     bool      `short:"j" long:"json" description:"Use JSON output format"`
        SizeOnly    bool      `short:"Z" long:"size-only" description:"Scan using sizes instead of checksums"`
    } `group:"Check ROM (-c, --chkrom) Options"`

    FixRom struct {
        Sources     []string  `short:"s" long:"src" description:"ROM source directory"`
        CreateDir   bool      `short:"D" long:"default-dir" description:"Create new machines as directories instead of zips"`
        SkipScan    bool      `short:"S" long:"skip-scan" description:"Skip ROM source directory scan"`
        ExtraTrash  bool      `short:"E" long:"extra-trash" description:"Move extra files to the trash"`
    } `group:"Fix ROM (-f, --fixrom) Options"`

    ChkTor struct {
        NoValid     bool      `long:"no-valid" description:"Do not validate torrent pieces"`
    } `group:"Check Torrent (-t, --chktor) Options"`

    LsTor struct {
        NoSize      bool      `long:"no-size" description:"Do not display torrent file sizes"`
    } `group:"List Torrent (-t, --lstor) Options"`

    TorZip struct {
        Force       bool      `long:"force" description:"Force TorrentZip conversion"`
    } `group:"TorrentZip (-z, --torzip) Options"`

    FltDat struct {
        Name        []string  `long:"name" description:"Machine name"  value-name:"REGEX"`
        Desc        []string  `long:"desc" description:"Machine description"  value-name:"REGEX"`
        Manu        []string  `long:"manu" description:"Machine manufacturer"  value-name:"REGEX"`
        Year        []string  `long:"year" description:"Machine year"  value-name:"REGEX"`
        Cat         []string  `long:"category" description:"Machine category"  value-name:"REGEX"`
        Invert      bool      `long:"invert" description:"Invert the filter"  value-name:"REGEX"`
    }  `group:"Filter DAT (-f.--fltdat) Options"`

    Dir2Dat struct {
        Name        string    `long:"name" description:"Name of DAT file" value-name:"STRING"`
        Desc        string    `long:"desc" description:"Description of DAT file" value-name:"STRING"`
    } `group:"Dir2Dat (-d, --dir2dat) Options" namespace:"dat"`

    FuzzyMv struct {
        DryRun      bool      `long:"dry-run" description:"Dry run without moving files"`
        Ratio       int       `long:"ratio" description:"Minimum match ratio (0-100)"`
        Confirm     bool      `long:"confirm" description:"Confirm each rename operation"`
        Match       string    `long:"match" description:"Directory containing file names to fuzzy match" value-name:"PATH"`
        Rename      string    `long:"rename" description:"Directory containing files to rename" value-name:"PATH"`
    } `group:"Fuzzy Rename (-m, --fuzzymv) Options"`

    GoRomDB struct {
        Dump        bool      `long:"dump" description:"Dump the contents of the database"`
        Scan        bool      `long:"scan" description:"Scan the current directory"`
        Lookup      string    `long:"lookup" description:"Look up a checksum"`
    } `group:"Database (-G, --goromdb) Options"`
}

var options Options

const help string = `[OPTIONS] [ARGS ...]

GoROM is a utility to manage emulator files. It includes operations to:
  * Check ROM sets versus a DAT file (-c, --chkrom)
  * Fix ROM sets to match a DAT file (-f, --fixrom)
  * Check if files match a BitTorrent file (-t, --chktor)
  * List the contents of a BitTorrent file (-l, --lstor)
  * Convert zip files into TorrentZip format (-z, --torzip)
  * Generate a DAT file for a directory (-d, --dir2dat)
  * Filter a DAT file on its data fields (-F, --fltdat)
  * Fuzzy rename files to match those in another directory (-m, --fuzzymv)
  * Manage the GoROM database (-G, --goromdb)

One and only one operation must be specified on the command line. See below
for the OPTIONS and ARGS specific to each operation. The Application Options
section lists OPTIONS used by more than one command.

GoROM uses ANSI color text display by default. If this causes issues on your
terminal, then you can disable it with the -C,--no-color option.

For operations involving ROM files, GoROM uses a database stored in each
directory to save the SHA-1 checksums and file modification times to speed up
subsequent operations. The database file is named .gorom.db.

Check ROM (-c, --chkrom
------------------------
Verifies if the ROMs in the current directory match a DAT file. By default,
SHA-1 checksums are used to guarantee file integrity but the file sizes can
optionally be used instead to speed up the process. The drawback is that
corrupt files of the same name and size are not detected and files with bad
names cannot be matched.

You can specify specific machines to check by specifying them as ARGS after the
OPTIONS. If no machines are specified, then all machines in the current
directory are checked.

Fix ROM (-f, --fixrom)
----------------------
Fixes the ROMs in the current directory to match a DAT file by renaming files
with bad names and copying missing or corrupt files from a set of source
directories. This is useful to apply update patches, fix an old or incomplete
ROM set, or to convert between different types of ROM sets like merged and
split.

Fixrom always uses SHA-1 checksums to determine the files to use. When started,
fixrom will scan the ROMs in the current directory and the specified source
directories and store the checksums into the database.

Fixrom will NEVER delete the original files and will instead move them to a
.trash directory. If you need to restore a ROM set back to its original state,
then you can simply move the contents of the .trash directory up one directory
level.

Fixrom automatically detects if a machine in a ROM set is stored as a directory
or a zip file and makes the fixed machine the same. For missing machines,
fixrom creates a zip file by default but this can be overriden with an option
to create a directory instead.

You can specify specific machines to fix by specifying them as ARGS after the
OPTIONS. If no machines are specified, then all machines in the current
directory are fixed.

Check Torrent (-t, --chktor)
----------------------------
Verifies the files in the current directory match a torrent. The names, sizes,
and checksums of the files are checked. This is similar to how torrent clients
validate a torrent prior to joining it. Extraneous files that do not belong to
the torrent are also listed. Chktor will not pad or otherwise alter the
contents of the files and will simply generate a report on the problems.

List Torrent (-l, --lstor)
--------------------------
Lists the contents of the given torrent file.

TorrentZip (-z, --torzip)
-------------------------
Converts regular zip files into TorrentZip files.  TorrentZip is a spec for zip
files that standardizes the central directory and compression method so that a
TorrentZip created with the same files is identical byte for byte regardless of
the platform that created it.  This allows for easier sharing of Zip files in a
torrent.

The zip files to convert are specified in ARGS after the OPTIONS. Torzip
replaces zip files with their TorrentZip equivalents.  Files that are already
TorrentZip are skipped.

Dir2Dat (-f, --dir2dat)
-----------------------
Generates a DAT file based on the contents of the current directory. Zip files
and subdirectories in the current directory are assumed to be the machines that
contain the ROM sets. Other types of files are silently skipped.

Filter DAT (-F, --fltdat)
-------------------------
Applies regular expressions to the fields of a DAT file to produce another DAT
file containing only the matches. The regular expression syntax used is RE2
(See https://github.com/google/re2/wiki/Syntax), which is similar to other
regular expression syntaxes like PCRE and Perl. Filter options of different
types are logically AND'ed together. Filter options of the same type are
logically OR'ed together.

Fuzzy Rename (-m, --fuzzymv)
----------------------------
Renames the files in one directory to their closest fuzzy matches in another
directory ignoring while preserving the file extensions. This is useful for
emulator front-ends that need the names of snapshots, covers, and other media
files to exactly match the ROM names.

For every file in the rename directory, fuzzymv compares it to each name in the
match directory to calculate a similarity score using the Levenshtein distance
of the sorted words. Fuzzymv then iterates through all the files to find the
best matches based on their scores. You can control the minimum score necessary
for fuzzymv to consider a match as valid which allows you to control the
strictness of the fuzzy match.

GoROM Database (-G, --goromdb)
------------------------------
Provides some utilities for managing and troubleshooting the ROM database of
SHA-1 checksums and modification times used by the ROM operations.
`

const examples string = `Examples:

* Check ROM set for errors
    gorom --chkrom "datfiles/MAME 0.220 ROMs (merged).xml"
* Fast check ROM set for errors (no checksums)
    gorom --chkrom "datfiles/MAME 0.220 ROMs (merged).xml" --size-only
* Limit ROM check to specific machines
    gorom --chkrom "datfiles/MAME 0.220 ROMs (merged).xml" puckman.zip mpatrol.zip asteroid.zip
* Check multimedia files for errors
    gorom --chkrom "datfiles/pS_AllProject_20200531_(cm).dat"
* Delete all extraneous files in a ROM set
    gorom --chkrom "datfiles/MAME 0.220 ROMs (merged).xml" --json | jq .extras[] | xargs rm
* Move ROMS with errors to a directory
    gorom --chkrom "datfiles/MAME 0.220 ROMs (merged).xml" --json > result
    jq '.machines[]|select(.status=="errors")|.path' result | xargs -i mv {} errors/
* Update an existing ROM set with an update set
    gorom --fixrom "../MAME 0.221 ROMs (merged).xml" --src "../MAME - Update ROMs (v0.220 to v0.221)" 
* Create a split ROM set from a merged set
    gorom --fixrom "../MAME 0.220 ROMs (split).xml" --src "datfiles/MAME 0.220 ROMs (merged)" 
* Create a 1G1R ROM set
    gorom --fixrom "../Atari - 2600 1G1R.dat" --src "../Atari - 2600 Roms"
* Update multimedia files
    gorom --fixrom "pS_MAME_AllProject_20200531_(cm).dat" --src "../MAME - Update EXTRAs (v0.220 to v0.221)" 
* Filter a DAT with only 1980's Pac-Man games
    gorom --fltdat "../datfiles/MAME 0.221 ROMs (merged).xml" --year '198[0-9]' --desc '(?i)pac[- ]man' > pacman.dat
* Make snapshot file names exactly match rom file names
    gorom --fuzzymv --match roms/ --rename snaps/
* Convert Zips to TorrentZip
    gorom --torzip *.zip
* Check if files match a torrent
    gorom --chktor "torrents/MAME 0.220 ROMs (split).torrent"
* List the torrent contents
    gorom --lstor "torrents/MAME 0.220 ROMs (split).torrent"
`

func usage(message string) {
    log.Println(message)
    fmt.Fprintf(os.Stderr, "Try '%s --help' for more information.\n", os.Args[0])
    os.Exit(1)
}

func main() {
    log.SetFlags(0)
    log.SetPrefix(fmt.Sprintf("%s: ", os.Args[0]))

    parser := flags.NewParser(&options, flags.HelpFlag | flags.PassDoubleDash)
    parser.Usage = help
    parser.NamespaceDelimiter = "-"

    args, err := parser.Parse()
    if err != nil {
        if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
            fmt.Println(err.Error())
            fmt.Println(examples)
            os.Exit(0)
        }
        usage(err.Error())
    }

    gorom.Progress = !options.App.NoProgress

    ops := parser.Command.Group.Find("Operations")
    if ops == nil {
        panic("No operations group")
    }
    opCount := 0
    for _, opt := range ops.Options() {
        if opt.IsSet() {
            opCount++
        }
    }
    if opCount == 0 {
        usage("No operation specified")
    }
    if opCount > 1 {
        usage("Only one operation allowed at a time")
    }

    if options.App.NoColor {
        term.IsTerminal = false
    }

    gorom.Progress = !options.App.NoProgress

    gorom.SignalInit(func() { term.CursorShow() });

    term.Init()
    term.CursorHide()

    ok := true
    if options.Operations.ChkRom != "" {
        datFile := filepath.ToSlash(options.Operations.ChkRom)
        ok, err = chkrom(datFile, args[:])
    }
    if options.Operations.FixRom != "" {
        datFile := filepath.ToSlash(options.Operations.FixRom)
        ok, err = fixrom(datFile, args[:], options.FixRom.Sources)
    }
    if options.Operations.ChkTor != "" {
        torrent := filepath.ToSlash(options.Operations.LsTor)
        ok, err = chktor(torrent)        
    }
    if options.Operations.LsTor != "" {
        torrent := filepath.ToSlash(options.Operations.LsTor)
        err = lstor(torrent)        
    }
    if options.Operations.TorZip {
        zipFiles := gorom.ToSlash(args)
        err = torzipFiles(zipFiles)
    }
    if options.Operations.Dir2Dat {
        err = dir2dat(args[:])
    }
    if options.Operations.FltDat != "" {
        datFile := filepath.ToSlash(options.Operations.FltDat)
        err = fltdat(datFile)
    }
    if options.Operations.FuzzyMv {
        if options.FuzzyMv.Match == "" || options.FuzzyMv.Rename == "" {
            usage("Fuzzy rename requires both --match and --rename options")
        }
        matchDir := filepath.ToSlash(options.FuzzyMv.Match)
        renameDir := filepath.ToSlash(options.FuzzyMv.Rename)
        err = fuzzymv(matchDir, renameDir)
    }
    if options.Operations.GoRomDB {
        err = goromdb()
    }

    term.CursorShow()

    if err != nil {
        log.Fatal(err)
    }
    if !ok {
        os.Exit(1)
    }
}
