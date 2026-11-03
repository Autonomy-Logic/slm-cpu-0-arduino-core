# SLM-CPU-0 bootloader upload contract

This is the contract between `slm-cpu-0-netflash` and the network bootloader that SLM-CPU-0 units ship
with from the factory. The bootloader is maintained and built by FACTS Engineering; this repository
neither builds nor changes it. Everything below describes the bootloader as implemented in the FACTS
SLM-MX repository, branch `feature/openplc-controller`, commit `9abbc51`, under `OpenPLC/`. Each
section names the source files it comes from.

Constants are written with the names used in this repository's copy of the shared header,
`overlay/variants/SLM_CPU_0/boot_shared.h` (prefix `SLM_CPU_0_`). The FACTS header uses the prefix
`SLMMX_` with the same values. `tools/slm-cpu-0-netflash/internal/layout/layout_test.go` checks the
uploader's constants against this repository's copy.

## Flash layout

Source: `common/boot_shared.h`.

| Address | Sectors | Size | Content |
| --- | --- | --- | --- |
| `0x08000000` | 0-1 | 32 KB | Network bootloader (write-protected) |
| `0x08008000` | 2 | 16 KB | Identity: backplane key and MAC records (write-protected, written only by the bootloader) |
| `0x0800C000` | 3 | 16 KB | Config log: last network settings and the current app record |
| `0x08010000` | 4-7 | 448 KB | OpenPLC application |

RAM is `0x20000000` to `0x20020000`. An upload writes only sectors 4 to 7 and the app record in
sector 3.

## Entering the bootloader from a running program

Sources: `arduino-core/overlay/variants/SLM_MX/slmmx_port.c` (application side),
`common/boot_shared.h`, the OpenPLC Editor's `resources/sources/Baremetal/modbus_types.h` and
`modbus_debug.cpp` (function codes).

- The uploader connects to the running OpenPLC program on Modbus TCP port 502, unit id 0.
- Request: function code `0x4C` followed by the 4 magic bytes `B0 07 10 AD`.
- The program answers `4C 7E` (success) and then resets. Before resetting, it writes a mailbox to the
  RTC backup registers: magic `0x534C4D42`, reason `1` (upload), its IP, mask and gateway, and a check
  value. Status `0x6C` means refused because a programming lock is engaged; `0x81` means wrong magic.
- `0x47` returns the runtime version (`7E` plus the version string) and `0x48` the device id
  (`7E`, length, bytes). The uploader uses both to confirm a new program is running.

## Modbus TCP in the bootloader

Source: `bootloader/src/service.c` (`modbus_answer`, `net_tcp_input`).

The bootloader listens on port 502 so the editor's Connect gets an answer:

- `0x4C` with the magic: `4C 7E` (already in the bootloader); wrong magic: `4C 81`; too short:
  exception `0x03`.
- Every other function code, `0x48` included: exception `0x01` (illegal function). The editor reads
  this as "No Firmware Detected".

## TFTP service

Sources: `bootloader/src/service.c`, `common/boot_shared.h`.

- UDP port 69. Transfers are answered from port 69 itself, not from a new transfer ID.
- Octet mode only. Requests are matched case-insensitively.
- One transfer at a time. A different client is refused with `busy with another transfer` until the
  current transfer has been idle for 3 s.
- The server retransmits its last packet after 1 s, up to 6 times, then drops the transfer.

### `firmware.bin` (write)

- Options: `blksize` (clamped to 8..1468, default 512), `tsize` (image size in bytes) and `crc32`
  (8 hex digits, the standard CRC-32 of the whole image). The bootloader answers with an OACK that
  echoes the options it received, or with `ACK 0` when none were sent.
- The image is checked before it is committed: no larger than 448 KB, at least 8 bytes, size equal to
  `tsize` when given, CRC-32 equal to `crc32` when given, and a plausible vector table (initial stack
  pointer word-aligned in RAM `0x20000000..0x20020000` or CCM RAM `0x10000000..0x10010000`; reset
  handler a Thumb address inside the image, above `0x08010008`). Source:
  `bootloader/src/app_image.c`.
- The first 1024 bytes are held in RAM and programmed last, so an interrupted upload never leaves a
  valid vector table.
- A refusal is a TFTP ERROR with a text message, for example `image CRC does not match the crc32
  option` or `not an SLM-MX application: bad vector table`.
- After the last block is acknowledged, the bootloader waits 1 s for a repeated last block, then
  resets into the new program.

### `info` (read)

One DATA block of at most 511 bytes, options declined, `key=value` lines:

| Key | Value |
| --- | --- |
| `device` | `SLM-MX` on current factory bootloaders |
| `mode` | `bootloader` |
| `bootloader` | bootloader version |
| `entry` | why it is running: `request`, `no_app`, `crash_loop` or `start` |
| `mac` | MAC address |
| `mac_source` | `identity` or `derived` |
| `ident_seq`, `hw_rev`, `serial` | present when an identity record exists |
| `ident_free` | free identity slots |
| `ip` | current IPv4 address |
| `dhcp` | `seeking`, `bound`, `fallback` or `off` |
| `app` | `valid` or `invalid` |
| `app_size`, `app_crc32` | the last committed image, when recorded |
| `key` | `present` or `absent` (backplane key) |
| `protect` | `on` or `off` (flash protection) |

The uploader accepts the bootloader when `mode=bootloader` and `device` is `SLM-MX` or `SLM-CPU-0`.

### `identity.bin` (write)

A 64-byte factory identity record. Factory tooling only; the uploader never sends it.

## Discovery

Source: `bootloader/src/service.c` (`discovery_reply`).

On UDP port 33333, a datagram starting with `OPENPLC_DISCOVER_V1` gets a JSON reply:
`{"service":"openplc-runtime","hostname":"SLM-MX","device":"SLM-MX (bootloader)","mac":"...",
"runtime_version":"bootloader","api_port":502}`. The editor's Search lists the board under that name.

## When the bootloader stays, and where it listens

Sources: `bootloader/src/main.c`, `bootloader/src/boot_policy.c`, `bootloader/src/dhcp.c`,
`common/boot_shared.h`.

- It stays when the program asked for it (mailbox, `entry=request`), when there is no valid program
  (`no_app`), or after 3 consecutive watchdog resets that the program did not clear within 30 s of
  running (`crash_loop`). Otherwise it starts the program.
- With a valid program, it returns to the program after 60 s without network activity and with no
  transfer in progress.
- Address: the one handed over in the mailbox (and remembered in the config log for the next power
  cycle); else the last network settings in the config log, using DHCP when they say so; else the
  identity record's address; else DHCP. When DHCP gets no answer, it falls back to `169.254.10.10`,
  mask `255.255.0.0`.
