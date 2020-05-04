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
package term

import (
    "bytes"
    "fmt"
    "io"
    "os"

    "golang.org/x/crypto/ssh/terminal"
    "github.com/eiannone/keyboard"
)

var (
    IsTerminal = terminal.IsTerminal(int(os.Stdout.Fd()))
    TerminalWidth, TerminalHeight, _ = terminal.GetSize(int(os.Stdout.Fd()))

    writer io.Writer = os.Stdout
    capture bytes.Buffer = bytes.Buffer{}
)

func CaptureStart() {
    IsTerminal = false
    writer = &capture
    capture.Reset()
}

func CaptureStop() []byte {
    writer = os.Stdout
    return capture.Bytes()
}

func Print(a ...interface{}) (int, error) {
    return fmt.Fprint(writer, a...)
}

func Printf(format string, a ...interface{}) (int, error) {
    return fmt.Fprintf(writer, format, a...)
}

func Println(a ...interface{}) (int, error) {
    return fmt.Fprintln(writer, a...)
}

func KeyPress() (rune, error)  {
    c, _, err := keyboard.GetSingleKey()
    return c, err
}

func CursorShow() {
    if IsTerminal {
        Print("\x1b[?25h")
    }
}

func CursorHide() {
    if IsTerminal {
        Print("\x1b[?25l")
    }
}

func Colorf(color int, format string, a ...interface{}) string {
    s := fmt.Sprintf(format, a...)
    if !IsTerminal {
        return s
    }

    return fmt.Sprintf("\x1b[38;5;%dm%s\x1b[39m", color, s)
}

func Black(format string, a ...interface{}) string {
    return Colorf(0, format, a...)
}

func Red(format string, a ...interface{}) string {
    return Colorf(1, format, a...)
}

func Green(format string, a ...interface{}) string {
    return Colorf(2, format, a...)
}

func Yellow(format string, a ...interface{}) string {
    return Colorf(3, format, a...)
}

func Blue(format string, a ...interface{}) string {
    return Colorf(4, format, a...)
}

func Magenta(format string, a ...interface{}) string {
    return Colorf(5, format, a...)
}

func Cyan(format string, a ...interface{}) string {
    return Colorf(6, format, a...)
}

func White(format string, a ...interface{}) string {
    return Colorf(7, format, a...)
}

func ClrEol() {
    if IsTerminal {
        Print("\x1b[0K")
    }
}
