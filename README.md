[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

# GoROM - Emulator ROM Management Utility

GoROM is a utility to manage emulator ROM files.  It consists of both a command-line interface (CLI) and a graphical user interface (GUI).  The CLI has more operations and is designed for power users.  The GUI is designed for simplicity and just covers the basic check and fix ROM functions.

In the emulator world, ROM sets are defined by DAT files which use an XML-based format that defines the ROM file names, checksums, and other metadata associated with the ROMs. GoROM supports both Clrmamepro and listxml styles of DAT files. GoROM also accepts gzipped DAT files (\*.gz) which it will decompress on the fly.

In DAT file parlance, related collections of ROMs are called machines. The ROMs for the machines are stored in either a zip file or a directory named the same as the machine. For each machine encountered in the DAT file, GoROM will automatically try to read either a directory or a zip file with the same name.

For operations involving ROM files, GoROM uses a bolt database stored in each directory to save the SHA-1 checksums and file modification times to speed up subsequent operations. The database file is named .gorom.db.

The GoROM CLI should run on any OS that has Go support. Note that the CLI uses ANSI escape codes for color output which are supported under most Linux terminals and in recent versions of Windows 10 command and power shells. Color output can be disabled with the -c,--no-color option if needed.

The GoROM CLI includes the following operations:

* **chkrom** - Verify the integrity of ROMs in a DAT file for their names, sizes and checksums
* **fixrom** - Fix or build a ROM set from a DAT file and a number of source directories
* **fltdat** - Filter a DAT file based on regular expressions applied to its data fields
* **dir2dat** - Create a DAT file from the files in the current directory
* **fuzzymv** - Rename files in one directory based on their closest fuzzy match to files in another directory
* **chktor** - Check that files match those in a torrent file and verify their integrity
* **lstor** - List the contents of a torrent file
* **torzip** - Convert a regular ZIP file to TorrentZip format
* **goromdb** - Manage the GoROM database

The GoROM GUI uses the Sciter engine (https://sciter.com/) which is only supported on Linux GTK, Windows, and OS X.

![GoROM GUI](https://filedn.com/lEnyCKkGcSaQKW9xHTReWxV/gorom.png)

## Installation

Download a release and copy the files to somewhere on your path.

## Build

From the top level directory, run make:

    $ make

The executable files are placed in the bin/ directory. Unit tests are available:

    $ make test

## Example Use Cases

Let's start out with some example use cases to demonstrate what GoROM can do.

### Check ROM set for errors
    $ gorom --chkrom "datfiles/MAME 0.220 ROMs (merged).xml"

### Fast check ROM set for errors (no checksums)
    $ gorom --chkrom "datfiles/MAME 0.220 ROMs (merged).xml" --size-only

### Check specific machines' ROM sets for errors
    $ gorom --chkrom "datfiles/MAME 0.220 ROMs (merged).xml" puckman.zip mpatrol.zip asteroid.zip

### Check multimedia files for errors
    $ gorom --chkrom "datfiles/pS_AllProject_20200531_(cm).dat"

### Delete all extraneous files in a ROM set
    $ gorom --chkrom "datfiles/MAME 0.220 ROMs (merged).xml" --json | jq .extras[] | xargs rm

### Move ROMS with errors to a directory
    $ gorom --chkrom "datfiles/MAME 0.220 ROMs (merged).xml" --json > result
    $ jq '.machines[]|select(.status=="errors")|.path' result | xargs -i mv {} errors/

### Update an existing ROM set with an update set
    $ gorom --fixrom "../MAME 0.221 ROMs (merged).xml" --src "../MAME - Update ROMs (v0.220 to v0.221)" 

### Create a split ROM set from a merged set
    $ gorom --fixrom "../MAME 0.220 ROMs (split).xml" --src "datfiles/MAME 0.220 ROMs (merged)" 

### Create a 1G1R ROM set
    $ gorom --fixrom "../Atari - 2600 1G1R.dat" --src "../Atari - 2600 Roms"

### Update multimedia files
    $ gorom --fixrom "pS_MAME_AllProject_20200531_(cm).dat" --src "../MAME - Update EXTRAs (v0.220 to v0.221)" 

### Filter a DAT with only 1980's Pac-Man games
    $ gorom --fltdat "../datfiles/MAME 0.221 ROMs (merged).xml" --year '198[0-9]' --desc '(?i)pac[- ]man' > pacman.dat

### Make snapshot file names exactly match rom file names
    $ gorom --fuzzymv --match roms/ --rename snaps/

### Convert Zips to TorrentZip
    $ gorom --torzip *.zip

### Check if files match a torrent
    $ gorom --lstor "torrents/MAME 0.220 ROMs (split).torrent"

### And many more...

## Motivation

There are many ROM management programs like Clrmamepro and RomCenter but the problem for me is that almost all are Windows GUIs. My daily driver is a Linux system and I wanted a simple command-line utility where I didn't have to fool with Windows emulation (i.e. WINE) on Linux.  All the ROM management programs I tried under WINE crashed constantly and looked very ugly.

GoROM started out as a few simple shell and Python scripts to help me manage my ROMS that quickly evolved into full-fledged programs. The reason I chose the Go language was due to Go's excellent parallel processing support that allows GoROM to perform operations quickly.

## chkrom

Chkrom takes a DAT file and verifies the ROMs in the current directory match the DAT file data. By default, SHA-1 checksums are used to guarantee file integrity but the file sizes can be optionally used instead to speed up the process. The drawback is that files of the same name and size could be corrupt and not detected.

You can specify specific machines to check by specifying them after the DAT file. If no machines are specified, then all machines in the current directory are checked.

By default, chkrom outputs an ANSI color text display listing the results. There are several options that suppress different parts of the output if desired. Chkrom can also output a JSON representation of the results that make it easier to do post-processing with uilities like jq.

All checksums generated by chkrom are inserted into a bolt database in the local directory named .gorom.db. When chkrom or other utilities subsequently run in the directory, the checksums from the database are used for each file whose modification time has not changed.

Example output:

    $ gorom --chkrom "../MAME 0.220 ROMs (merged).xml"
    MAME 0.220 ROMs (merged)
    Parsing DAT file...
    1on1gov.zip : ROM ERRORS
      1on1.u119 : BAD NAME (1on1.bin)
      1on1.u120 : OK
      at28c16 : OK
      mg10 : OK
      ooo-0.u0217 : MISSING
      ooo-1.u0218 : OK
      ooo-2.u0219 : OK
      ooo-3.u0220 : OK
      ooo-4.u0221 : CORRUPT
      ooo-5.u0222 : OK
      ooo-6.u0223 : OK
      ooo-7.u0323 : OK
      ooo-7.u0324 : EXTRA
    2mindril.zip : OK
      d58-08.ic27 : OK
      d58-09.ic28 : OK
      d58-10.ic29 : OK
      d58-11.ic31 : OK
      d58-37.ic9 : OK
      d58-38.ic11 : OK
    3b1 : MISSING
    :
    Machine Stats
      All OK          : 72 (85.7%)
      ROMs Corrupt    : 5 (6.0%)
      ROMs Bad Name   : 3 (3.6%)
      ROMs Missing    : 6 (7.1%)
      ROMs Extra      : 4 (4.8%)
      Machine Missing : 1 (1.2%)
      Machine Corrupt : 0 (0.0%)
      Total Machines  : 84
      Extra Files     : 1
    
    ROM Stats
      OK        : 1303 (73.3%)
      Corrupt   : 137 (7.7%)
      Bad Name  : 26 (1.5%)
      Missing   : 312 (17.5%)
      Total     : 1778
      Extra     : 5

An example of all output suppressed except for machines with errors:

    $ chkrom --chkrom "../MAME 0.220 ROMs (merged).xml" -c -o -r -H -e -p 
    1on1gov.zip : ROM ERRORS
    3b1 : MISSING
    pucman : CORRUPT
    :

## fixrom

Fixrom makes the ROMs in the current directory match a DAT file by renaming files with bad names and copying missing or corrupt files from a set of source directories. This is useful to apply update patches, fix an old or incomplete ROM set, or to convert between different types of ROM sets like merged and split.

Fixrom always uses SHA-1 checksums to determine the files to use. When started, fixrom will scan the ROMs in the current directory and the specified source directories. Generated checksums are added to a bolt database so subsequent runs are much faster and will look at the modification times of files to determine if they need new checksums.

Fixrom will **NEVER** delete the original files and will instead move them to the .trash directory. If you need to restore a ROM set back to its original state, then you can simply move the contents of the .trash directory up one directory level.

Example output:

    $ gorom --fixrom "../MAME 0.221 ROMs\ (merged).xml" --no-ok --srv "../MAME - Update ROMs (v0.220 to v0.221)" 
    Scanning directory .
    Scanning directory ../MAME - Update ROMs (v0.220 to v0.221)
    MAME 0.221 ROMs (merged)
    airwlkrs.zip : FIXING
      mpr-19236.10 : COPY from airwlkrs.zip
      mpr-19237.11 : COPY from airwlkrs.zip
      OK
    aligator.zip : FIXING
      aligatorp/a0.bin : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/aligatorp.zip
      aligatorp/a1.bin : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/aligatorp.zip
      aligatorp/a2.bin : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/aligatorp.zip
      aligatorp/a3.bin : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/aligatorp.zip
      aligatorp/a4.bin : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/aligatorp.zip
      aligatorp/a5.bin : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/aligatorp.zip
      aligatorp/a6.bin : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/aligatorp.zip
      aligatorp/a7.bin : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/aligatorp.zip
      aligatorp/all_27-10_notext.u44 : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/aligatorp.zip
      aligatorp/all_27-10_notext.u45 : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/aligatorp.zip
      aligatorp/b0.bin : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/aligatorp.zip
    :
    xmen.zip : FIXING
      xmenu/065-ubb04.10d : RENAME from 065-ubb04.10d
      xmenu/065-ubb05.10f : RENAME from 065-ubb05.10f
      xmenu/xmen_ubb.nv : RENAME from xmen_ubb.nv
      065-eba04.10d : RENAME from xmene/065-eba04.10d
      065-eba05.10f : RENAME from xmene/065-eba05.10f
      xmen_eba.nv : RENAME from xmene/xmen_eba.nv
      xmenua/065-ueb04.10d : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/xmenua.zip
      xmenua/065-ueb05.10f : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/xmenua.zip
      xmenua/xmen_ueb.nv : COPY from ../MAME - Update ROMs (v0.220 to v0.221)/xmenua.zip
      OK
    Renaming temporary files
    
    Machine Stats
      OK     : 12937 (99.1%)
      Fixed  : 111 (0.9%)
      Failed : 5 (0.0%)
      Total  : 13053

## fltdat

Fltdat applies regular expressions to the fields of a DAT file to produce another DAT file containing only the matches. The regular expression syntax used is [RE2](https://github.com/google/re2/wiki/Syntax), which is similar to other regular expression syntaxes like PCRE and Perl. Filter options of different types are logically AND'ed together. Filter options of the same type are logically OR'ed together.

## dir2dat

Dir2dat generates a DAT file based on the contents of the current directory. Zip files and subdirectories in the current directory are assumed to be the machines that contain the ROM sets. Other types of files are skipped.

## fuzzymv

Fuzzymv renames the files in one directory to their closest fuzzy matches in another directory ignoring but preserving the file extensions. This is useful for emulator front-ends that need the names of snapshots, covers, and other media files to exactly match the ROM names.

For every file in the rename directory, fuzzymv compares it to each name in the match directory to calculate a similarity score using the Levenshtein distance of the sorted words. Fuzzymv then iterates through all the files to find the best matches based on their scores. You can control the minimum score necessary for fuzzymv to consider a match as valid which allows you to control the strictness of the fuzzy match.
  
Example output:

    $ gorom --fuzzymv --match roms --rename snaps
    Matches found: 1295/1306
    'Bottom of the 9th.jpg' => 'Bottom of the 9th (USA).jpg'
    'Warcraft II - The Dark Saga.jpg' => 'WarCraft II - The Dark Saga (USA) (En,Fr,De,Es,It).jpg'
    'Disney's Mulan Story Studio.jpg' => 'Disney's Story Studio - Mulan (USA).jpg'
    'Spec Ops - Covert Assault.jpg' => 'Spec Ops - Covert Assault (USA).jpg'
    'Tony Hawk's Pro Skater.jpg' => 'Tony Hawk's Pro Skater (USA).jpg'
    'Agile Warrior F-111X.jpg' => 'Agile Warrior - F-111X (USA).jpg'
    'Tekken.jpg' => 'Tekken (USA).jpg'
    'Rugrats - Totally Angelica.jpg' => 'Nickelodeon Rugrats - Totally Angelica (USA).jpg'
    'International Track & Field 2000.jpg' => 'International Track & Field 2000 (USA).jpg'
    :
    'F1 World Grand Prix 2000.jpg' => 'F1 World Grand Prix (USA).jpg'
    'FIFA 2001 - Major League Soccer.jpg' => 'FIFA 2001 (USA).jpg'
    'NBA Jam T.E..jpg' => 'NBA Jam - Tournament Edition (USA).jpg'
    'Street Fighter EX 2 Plus.jpg' => 'Street Fighter EX2 Plus (USA).jpg'
    'Risk.jpg' => 'Risk - The Game of Global Domination (USA).jpg'
    'NHL Powerplay 98.jpg' => 'NHL Powerplay 98 (USA) (En,Fr,De).jpg'

## chktor

Chktor verifies the names, sizes, and checksums of files in the current directory versus a supplied torrent file. This is similar to how torrent clients validate a torrent prior to joining it. Extraneous files that do not belong to the torrent are also listed. Chktor will not pad or otherwise alter the contents of the files and will simply generate a report on the problems.

Example output:

    $ gorom --chktor ../eXoDOS_v4.torrent
    Torrent Name : eXoDOS
    Announce     : udp://tracker.coppersurfer.tk:6969/announce
    Piece Length : 16777216 (16 MiB)
    Pieces       : 31459
    Files        : 7015
    Total Length : 527778579946 (491.532 GiB)
    
    Verifying files...
    !DOSmetadata.zip : OK
    Extras/DGI/DOS Game Installer.7z : OK
    Extras/DOS Game Installer Readme.txt : OK
    Extras/Linux Readme.txt : OK
    Extras/Linux/Linux.tar.gz : OK
    Extras/Linux/extract.sh : OK
    :

    Finding extra files...
    
    Torrent Stats
      Missing  : 0
      Bad Size : 0
      Extras   : 0
    
    Validating pieces...
    OK

## lstor

Lstor lists the metadata, file names and file sizes of a torrent file.

Example output:

    $ gorom --lstor eXoDOS_v4.torrent
    Torrent Name : eXoDOS
    Announce     : udp://tracker.coppersurfer.tk:6969/announce
    Piece Length : 16777216 (16 MiB)
    Pieces       : 31459
    Files        : 7015
    Total Length : 527778579946 (491.532 GiB)
    !DOSmetadata.zip 18977163117 (17.674 GiB)
    Extras/DGI/DOS Game Installer.7z 4970253 (4.74 MiB)
    Extras/DOS Game Installer Readme.txt 8557 (8.356 KiB)
    Extras/Linux Readme.txt 12296 (12.008 KiB)
    Extras/Linux/Linux.tar.gz 151389514 (144.376 MiB)
    Extras/Linux/extract.sh 1505 (1.47 KiB)
    LaunchBox.zip 540171660 (515.148 MiB)
    Setup.bat 13967 (13.64 KiB)
    XODOSMetadata.zip 19632008528 (18.284 GiB)
    _ReadMe_.txt 563 (563 B)
    eXoDOS Catalog.pdf 2173730 (2.073 MiB)
    eXoDOS Manual.pdf 1907093 (1.819 MiB)
    eXoDOS ReadMe.txt 24530 (23.955 KiB)
    eXoDOS/Games/$100,000 Pyramid (1988).zip 100892 (98.527 KiB)
    :

## torzip

Torzip converts regular zip files into TorrentZip files.  TorrentZip is a specification for zip files that standardizes the central directory and compression method so that a TorrentZip created with the same files is identical byte for byte regardless of the platform that created it.  This allows for easier sharing of Zip files in a torrent.

Torzip replaces zip files with their TorrentZip equivalents.  Files that are already TorrentZip are skipped.
