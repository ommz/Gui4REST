# Gui4REST
A no-bloat native GUI client for REST APIs

Back to the basics of what's essential in a REST API client:
- Light and light on its feet (~11 MB)
- Configurable yet stays out of your way
- Cross-platform
- Light & dark theming
- Compiles to a single statically-linked executable: zero external dependencies
- Zero embedded phone-home elements


## Screenshots

| Dark Theme  | Light Theme |
| ------------- | ------------- |
| ![Dark Theme](screenshot-dark-theme.png)  | ![Light Theme](screenshot-light-theme.png)  |

![Dark Theme](screenshot-dark-theme.png)



## Installing & Running
Portable executable binaries for Windows andLinux are available in the releases page.
MacOS & BSD builds are coming (one import proved quite a handful about native graphics drivers for the usual painless golang cross-compiles)

You may choose to run Gui4REST directly by double-clicking on the downloaded file.
On Linux, if double-clicking does not fire it up, make it executable first by running:
```
cd /path/to/Gui4REST
chmod +x Gui4REST
```


## TODO
- Burst Mode
- Saving & loading of previously run APIRequests
- Tests
