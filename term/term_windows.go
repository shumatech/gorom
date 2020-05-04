package term

import (
    "golang.org/x/sys/windows"
)

func Init() {
    if IsTerminal {
        h, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE);
        if err == nil {
            var mode uint32
            err = windows.GetConsoleMode(h, &mode)
            if err == nil {
                mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
                windows.SetConsoleMode(h, mode)
            }
        }
    }
}