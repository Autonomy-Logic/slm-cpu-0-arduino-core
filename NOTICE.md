# NOTICE

This repository builds the `FACTS:stm32` Arduino platform for the Synergy Logic
SLM-CPU-0 and its Ethernet uploader. It does not vendor the upstream sources:
`tools/ci/build-release.py` downloads them at the versions and SHA-256 digests
pinned in `tools/ci/upstream.json`, trims them and applies `overlay/`.

## Upstream sources

| Component | Version | Source | Licence |
| --- | --- | --- | --- |
| Arduino_Core_STM32 (STM32duino) | 2.12.0 | https://github.com/stm32duino/Arduino_Core_STM32 | Per-file headers (LGPL-2.1, BSD-3-Clause, Apache-2.0 for ST and Arm parts) |
| STM32Ethernet | 1.5.0 | https://github.com/stm32duino/STM32Ethernet | LGPL-2.1 |
| LwIP (STM32duino) | 2.2.1 | https://github.com/stm32duino/LwIP | BSD-3-Clause |
| xpack-arm-none-eabi-gcc | 14.2.1-1.1 | https://github.com/xpack-dev-tools/arm-none-eabi-gcc-xpack | GPL with the GCC runtime exception |
| CMSIS | 6.2.0 | https://github.com/stm32duino/ArduinoModule-CMSIS | Apache-2.0 |
| STM32Tools | 2.4.0 | https://github.com/stm32duino/Arduino_Tools | Per-file headers |

The toolchain, CMSIS and STM32Tools are referenced from the board manager
index at their original URLs and are not redistributed here. Upstream files
keep their original copyright and licence headers.

## Changes made by this repository

- The core is trimmed to the STM32F4 series and the STM32F407 device headers.
- `boards.txt`, the upload recipe (`overlay/platform.upload.txt`) and the
  `SLM_CPU_0` variant (`overlay/variants/SLM_CPU_0/`) are added. The variant
  pin arrays and linker script derive from the STM32duino generic
  F407V(E-G)T variant (BSD-3-Clause, STMicroelectronics).
- LwIP and STM32Ethernet are bundled as `SLM_CPU_0_LwIP` and
  `SLM_CPU_0_Ethernet`, with renamed headers, the MAC taken from the board's
  identity sector, a 20 s DHCP timeout, a link-local fallback address, a
  `macAddress()` alias, a public `EthernetServer::accept()` that hands each
  new connection to the caller once, `EthernetClient::availableForWrite()`,
  and an initialised flag in the accept callback.
- `platform.txt` drops the debugger and CMSIS-DSP references and uses the
  short tool name `arm-gcc`.
- The uploader `slm-cpu-0-netflash` (`tools/slm-cpu-0-netflash/`) and the
  variant's boot and flash-layout files are derived from FACTS Engineering's
  SLM-CPU-0 OpenPLC work and published under GPL-3.0 with their approval.

The slot backplane driver for SLM I/O modules is proprietary to FACTS
Engineering and is not part of this repository or its releases.
