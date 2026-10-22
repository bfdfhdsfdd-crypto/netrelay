# NetRelay GUI

The desktop application uses the NetRelay product name and release version in
its window title. Its app icon is defined by `build/appicon.png` and
`build/windows/icon.ico`.

The log panel batches incoming messages and retains a bounded 200,000-character
history so high-volume sessions do not continuously re-highlight an unbounded
log buffer.
