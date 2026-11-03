# SLM-CPU-0 Arduino core (`FACTS:stm32`)

Arduino platform and Ethernet uploader for the Synergy Logic SLM-CPU-0
(STM32F407VE, LAN8742 Ethernet, 8 SLM I/O slots). The OpenPLC Editor installs
it through the board manager when the SLM-CPU-0 package is installed.

| | |
| --- | --- |
| Board manager URL | `https://raw.githubusercontent.com/Autonomy-Logic/slm-cpu-0-arduino-core/main/package_FACTS_slm_cpu_0_index.json` |
| FQBN | `FACTS:stm32:slm_cpu_0` |
| Upload | Ethernet, through the board's network bootloader (`slm-cpu-0-netflash`) |
| Licence | GPL-3.0 (see `LICENSE` and `NOTICE.md`) |

## Install and use

```
URL=https://raw.githubusercontent.com/Autonomy-Logic/slm-cpu-0-arduino-core/main/package_FACTS_slm_cpu_0_index.json
arduino-cli core update-index --additional-urls "$URL"
arduino-cli core install FACTS:stm32 --additional-urls "$URL"
arduino-cli compile -b FACTS:stm32:slm_cpu_0 --export-binaries <sketch>
arduino-cli upload  -b FACTS:stm32:slm_cpu_0 -p <board IP> <sketch>
```

The upload asks the running OpenPLC program to start the bootloader (Modbus
function 0x4C), sends the image over TFTP and checks that the new program
answers. A board without a program waits in its bootloader and is uploaded to
directly. The protocol is in [docs/bootloader-contract.md](docs/bootloader-contract.md).
The board menu also offers SWD upload with STM32CubeProgrammer, for boards
without read protection.

Sketches reach the board services through `<slm_cpu_0.h>` (bootloader
hand-over, crash-loop counter, network settings for the bootloader) and the
flash layout through `<boot_shared.h>`.

## Layout

| Path | Purpose |
| --- | --- |
| `overlay/boards.txt` | the `slm_cpu_0` board |
| `overlay/platform.upload.txt` | upload recipes appended to the STM32duino `platform.txt` |
| `overlay/variants/SLM_CPU_0/` | pins, clock, linker script, `<Ethernet.h>` shims, bootloader interface |
| `tools/slm-cpu-0-netflash/` | the Ethernet uploader (Go) |
| `tools/ci/build-release.py` | builds the platform archive, uploader archives and index |
| `tools/ci/upstream.json` | pinned upstream archives and toolchain downloads |
| `tools/ci/build-uploader.sh` | cross-compiles the uploader for every index host |
| `tools/ci/localize-index.py` | points an index at a local server for pre-release tests |
| `tools/ci/verify-index.py` | checks every asset of a published index |
| `tools/ci/sketches/slm_cpu_0_smoke/` | compile check for the platform |
| `tools/ci/RENAMES.md` | identifier mapping from the SLM-MX proof of concept |

## Build a release locally

Needs Python 3.12+ and Go.

```
tools/ci/build-uploader.sh dist/tools 0.1.0
tools/ci/build-release.py --version 0.1.0 --out dist
```

`dist/` then holds `FACTS-stm32-slm-cpu-0-0.1.0.tar.bz2`, one
`slm-cpu-0-netflash-0.1.0-<host>.tar.gz` per host and
`package_FACTS_slm_cpu_0_index.json`, with URLs pointing at the GitHub release
`v0.1.0`. Downloads are cached in `build/`. `--no-netflash` builds a
compile-only package; `--refresh-tools` refreshes the toolchain entries in
`upstream.json` from STMicroelectronics' index.

To install the result before publishing it, serve `dist/` and repoint a copy
of the index:

```
(cd dist && python3 -m http.server 8765 --bind 127.0.0.1) &
cp dist/package_FACTS_slm_cpu_0_index.json /tmp/index-local.json
tools/ci/localize-index.py /tmp/index-local.json http://127.0.0.1:8765
arduino-cli core update-index --additional-urls file:///tmp/index-local.json
arduino-cli core install FACTS:stm32@0.1.0 --additional-urls file:///tmp/index-local.json
arduino-cli compile -b FACTS:stm32:slm_cpu_0 tools/ci/sketches/slm_cpu_0_smoke
```

## Branches

Work lands on `development` through pull requests; releases are merged to
`main` and tagged `v<version>`.
