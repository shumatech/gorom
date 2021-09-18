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
    "os"
    "path"
    "regexp"
    "sort"
    "strings"

    "gorom/util"
    "gorom/romio"
    "gorom/term"

    "github.com/paul-mannino/go-fuzzywuzzy"
)

var (
    dirNameRe = regexp.MustCompile(`[^0-9a-z ]`)
)

const (
    MatchLimit = 10
)

type match struct {
    key string
    score int
}

func dirMap(dir string, useBase bool) (*util.StringBiMap, error) {
    fh, err := os.Open(dir)
    if err != nil {
        return nil, err
    }

    infos, err := fh.Readdir(0)
    if err != nil {
        return nil, err
    }

    fh.Close()

    dmap := util.NewStringBiMap()

    for _, info := range infos {
        if !info.IsDir() {
            // Remove the file extensions
            name := info.Name()
            base := strings.TrimSuffix(name, path.Ext(name))
            // Convert underscores and commas to spaces
            key := strings.ReplaceAll(base, "_", " ")
            key = strings.ReplaceAll(key, ",", " ")
            // Convert to lower case
            key = strings.ToLower(key)
            // Remove non-word/non-space characters
            key = dirNameRe.ReplaceAllString(key, "")
            // Split and recombine to remove duplicate space
            key = strings.Join(strings.Fields(key), " ")

            if useBase {
                dmap.Set(key, base)
            } else {
                dmap.Set(key, name)
            }
        }
    }

    return dmap, nil
}

func matchAdd(l []match, m *match) []match {
    l = append(l, *m)

    sort.Slice(l, func(i, j int) bool {
        if l[i].score == l[j].score {
            if len(l[i].key) == len(l[j].key) {
                return l[i].key < l[j].key
            } else {
                return len(l[i].key) > len(l[j].key)
            }
        } else {
            return l[i].score > l[j].score
        }
    })

    if len(l) > MatchLimit {
        l = l[:MatchLimit]
    }

    return l
}

func fuzzymv(matchDir, renameDir string) error {
    matches, err := dirMap(matchDir, true)
    if err != nil {
        return err
    }
    renames, err := dirMap(renameDir, false)
    if err != nil {
        return err
    }

    // Remove exact filename matches
    for value, key := range renames.Values() {
        base := romio.MachName(value)
        if _, ok := matches.GetValue(base); ok {
            renames.Delete(key, value)
            matches.Delete(key, base)
        }
    }

    // Calculate the list of matches meeting the minimum ratio
    bestMatches := map[string][]match{}
    bestRenames := map[string][]match{}

    count := 1
    total := len(renames.Keys())
    for rkey, rfile := range renames.Keys() {
        util.Progressf("Matching (%d/%d): '%s'", count, total, rfile)

        for mkey := range matches.Keys() {
            setRatio  := fuzzy.TokenSetRatio(rkey, mkey)
            sortRatio := fuzzy.TokenSortRatio(rkey, mkey)

            if setRatio >= options.FuzzyMv.Ratio || sortRatio >= options.FuzzyMv.Ratio {
                score := setRatio + sortRatio
                bestMatches[mkey] = matchAdd(bestMatches[mkey], &match{rkey, score})
                bestRenames[rkey]= matchAdd(bestRenames[rkey], &match{mkey, score})
            }
        }
        count++
    }

    util.Progressf("")
    term.Println(term.Magenta("Matches found: %d/%d", len(bestRenames), len(renames.Keys())))

    // Display renames we didn't match anything to
    for rkey, rfile := range renames.Keys() {
        if _, ok := bestRenames[rkey]; !ok {
            term.Print(term.Red("'%s' => NO MATCH\n", rfile))
        }
    }

    // Iterate through the match lists expanding the match window every pass
    for limit := 0; limit < MatchLimit; limit++ {
        for mkey, mbest := range bestMatches {
            mfile, _ := matches.Get(mkey)

            window:
            for i := 0; i <= limit && i < len(mbest); i++ {
                rkey := mbest[i].key
                rbest := bestRenames[rkey]
                rfile, _ := renames.Get(rkey)

                for j := 0; j <= limit && j < len(rbest); j++ {
                    if rbest[j].key == mkey {
                        delete(bestMatches, mkey)
                        delete(bestRenames, rkey)

                        newfile := mfile + path.Ext(rfile)
                        term.Printf("'%s' => '%s'\n", rfile, newfile)

                        if options.FuzzyMv.Confirm {
                            term.Print("Rename [Y/N]? ");
                            char, err := term.KeyPress()
                            if (err != nil) {
                                panic(err)
                            }
                            term.Printf("%c\n", char)
                            if char != 'y' && char != 'Y' {
                                break window
                            }
                        }

                        if !options.FuzzyMv.DryRun {
                            err = os.Rename(path.Join(renameDir, rfile), path.Join(renameDir, newfile))
                            if err != nil {
                                term.Println(err)
                            }
                        }
                        break window
                    }
                }
            }
        }
    }

    for rkey := range bestRenames {
        rfile, _ := renames.Get(rkey)
        term.Print(term.Yellow("'%s' => LOST MATCH\n", rfile))
    }

    return nil
}
