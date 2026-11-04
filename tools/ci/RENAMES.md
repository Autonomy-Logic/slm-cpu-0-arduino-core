# SLM-MX to SLM-CPU-0 renames

Identifier mapping between the FACTS proof of concept (SLM-MX, commit 9abbc51) and this core. Names only: every value, layout and wire string is unchanged.

## Names

| Old | New |
| --- | --- |
| FQBN FACTS:stm32:slm_mx | FACTS:stm32:slm_cpu_0 |
| Board name SLM-MX | SLM-CPU-0 |
| Platform name FACTS SLM-MX (STM32F4) | FACTS SLM-CPU-0 (STM32F4) |
| build.board SLM_MX (define ARDUINO_SLM_MX) | SLM_CPU_0 (ARDUINO_SLM_CPU_0) |
| Variant folder variants/SLM_MX | variants/SLM_CPU_0 |
| variant_SLM_MX.h / variant_SLM_MX.cpp | variant_SLM_CPU_0.h / variant_SLM_CPU_0.cpp |
| Header <slmmx.h> | <slm_cpu_0.h> |
| slmmx_pins.c / slmmx_port.c | slm_cpu_0_pins.c / slm_cpu_0_port.c |
| common/boot_shared.h | variants/SLM_CPU_0/boot_shared.h (file name kept) |
| common/slmmx_boot.c | variants/SLM_CPU_0/slm_cpu_0_boot.c |
| Library SLMMX_LwIP (SLMMX_LwIP.h) | SLM_CPU_0_LwIP (SLM_CPU_0_LwIP.h) |
| Library SLMMX_Ethernet (SLMMX_Ethernet.h, SLMMX_EthernetUdp.h) | SLM_CPU_0_Ethernet (SLM_CPU_0_Ethernet.h, SLM_CPU_0_EthernetUdp.h) |
| Upload tool slmmx-netflash (recipe key slmmx_netflash) | slm-cpu-0-netflash (slm_cpu_0_netflash) |
| Index package_FACTS_slmmx_index.json | package_FACTS_slm_cpu_0_index.json |
| Platform archive FACTS-stm32-slmmx-<ver>.tar.bz2 | FACTS-stm32-slm-cpu-0-<ver>.tar.bz2 |
| Library SLMBackplane | not in this core; ships precompiled in the openplc-packages SLM-CPU-0 package |

## Macros, types and functions provided by the variant

Used by: B = SLMBackplane sources, H = FACTS HAL (vpp/hal/slm_mx.cpp).

| Old | New | Used by |
| --- | --- | --- |
| `SLMMX_APP_BASE` | `SLM_CPU_0_APP_BASE` |  |
| `SLMMX_APP_COMMIT_SIZE` | `SLM_CPU_0_APP_COMMIT_SIZE` |  |
| `SLMMX_APP_END` | `SLM_CPU_0_APP_END` |  |
| `SLMMX_APP_FIRST_SECTOR` | `SLM_CPU_0_APP_FIRST_SECTOR` |  |
| `SLMMX_APP_LAST_SECTOR` | `SLM_CPU_0_APP_LAST_SECTOR` |  |
| `SLMMX_APP_MAX_SIZE` | `SLM_CPU_0_APP_MAX_SIZE` |  |
| `slmmx_app_stable` | `slm_cpu_0_app_stable` | H |
| `SLMMX_APP_STABLE_MS` | `SLM_CPU_0_APP_STABLE_MS` | H |
| `SLMMX_BKP_BOOTCOUNT` | `SLM_CPU_0_BKP_BOOTCOUNT` |  |
| `SLMMX_BKP_CHECK` | `SLM_CPU_0_BKP_CHECK` |  |
| `SLMMX_BKP_GW` | `SLM_CPU_0_BKP_GW` |  |
| `SLMMX_BKP_IP` | `SLM_CPU_0_BKP_IP` |  |
| `SLMMX_BKP_MAGIC` | `SLM_CPU_0_BKP_MAGIC` |  |
| `SLMMX_BKP_MASK` | `SLM_CPU_0_BKP_MASK` |  |
| `SLMMX_BKP_REASON` | `SLM_CPU_0_BKP_REASON` |  |
| `slmmx_boot` | `slm_cpu_0_boot` |  |
| `SLMMX_BOOT_BASE` | `SLM_CPU_0_BOOT_BASE` |  |
| `SLMMX_BOOT_SHARED_H` | `SLM_CPU_0_BOOT_SHARED_H` |  |
| `SLMMX_BOOT_SIZE` | `SLM_CPU_0_BOOT_SIZE` |  |
| `SLMMX_CFG_APP` | `SLM_CPU_0_CFG_APP` |  |
| `slmmx_cfg_app_t` | `slm_cpu_0_cfg_app_t` |  |
| `slmmx_cfg_append` | `slm_cpu_0_cfg_append` |  |
| `SLMMX_CFG_BASE` | `SLM_CPU_0_CFG_BASE` |  |
| `SLMMX_CFG_DATA_SIZE` | `SLM_CPU_0_CFG_DATA_SIZE` |  |
| `SLMMX_CFG_ENTRY_SIZE` | `SLM_CPU_0_CFG_ENTRY_SIZE` |  |
| `slmmx_cfg_entry_t` | `slm_cpu_0_cfg_entry_t` |  |
| `slmmx_cfg_is_valid` | `slm_cpu_0_cfg_is_valid` |  |
| `slmmx_cfg_latest` | `slm_cpu_0_cfg_latest` |  |
| `SLMMX_CFG_MAGIC` | `SLM_CPU_0_CFG_MAGIC` |  |
| `SLMMX_CFG_NET` | `SLM_CPU_0_CFG_NET` |  |
| `slmmx_cfg_net_t` | `slm_cpu_0_cfg_net_t` | H |
| `SLMMX_CFG_SECTOR` | `SLM_CPU_0_CFG_SECTOR` |  |
| `SLMMX_CFG_SIZE` | `SLM_CPU_0_CFG_SIZE` |  |
| `SLMMX_CFG_SLOTS` | `SLM_CPU_0_CFG_SLOTS` |  |
| `SLMMX_CFG_TYPE_MAX` | `SLM_CPU_0_CFG_TYPE_MAX` |  |
| `SLMMX_CRASH_LOOP_LIMIT` | `SLM_CPU_0_CRASH_LOOP_LIMIT` |  |
| `slmmx_crc32` | `slm_cpu_0_crc32` |  |
| `SLMMX_DEFAULT_IP` | `SLM_CPU_0_DEFAULT_IP` |  |
| `SLMMX_DEFAULT_MASK` | `SLM_CPU_0_DEFAULT_MASK` |  |
| `slmmx_enter_bootloader` | `slm_cpu_0_enter_bootloader` | H |
| `SLMMX_ETHERNET_SHIM_H` | `SLM_CPU_0_ETHERNET_SHIM_H` |  |
| `SLMMX_ETHERNETUDP_SHIM_H` | `SLM_CPU_0_ETHERNETUDP_SHIM_H` |  |
| `SLMMX_FLASH_BASE` | `SLM_CPU_0_FLASH_BASE` |  |
| `SLMMX_FLASH_END` | `SLM_CPU_0_FLASH_END` |  |
| `SLMMX_FLASH_PTR` | `SLM_CPU_0_FLASH_PTR` | B |
| `slmmx_flash_sector_base` | `slm_cpu_0_flash_sector_base` |  |
| `slmmx_flash_sector_of` | `slm_cpu_0_flash_sector_of` |  |
| `slmmx_get_mac` | `slm_cpu_0_get_mac` |  |
| `SLMMX_H` | `SLM_CPU_0_H` |  |
| `SLMMX_IDENT_BASE` | `SLM_CPU_0_IDENT_BASE` |  |
| `slmmx_ident_is_valid` | `slm_cpu_0_ident_is_valid` |  |
| `SLMMX_IDENT_MAGIC` | `SLM_CPU_0_IDENT_MAGIC` |  |
| `slmmx_ident_newest` | `slm_cpu_0_ident_newest` |  |
| `SLMMX_IDENT_REC_BASE` | `SLM_CPU_0_IDENT_REC_BASE` |  |
| `SLMMX_IDENT_REC_SIZE` | `SLM_CPU_0_IDENT_REC_SIZE` |  |
| `SLMMX_IDENT_REC_SLOTS` | `SLM_CPU_0_IDENT_REC_SLOTS` |  |
| `slmmx_ident_seal` | `slm_cpu_0_ident_seal` |  |
| `SLMMX_IDENT_SECTOR` | `SLM_CPU_0_IDENT_SECTOR` |  |
| `SLMMX_IDENT_SIZE` | `SLM_CPU_0_IDENT_SIZE` |  |
| `SLMMX_IDENT_VERSION` | `SLM_CPU_0_IDENT_VERSION` |  |
| `slmmx_identity_t` | `slm_cpu_0_identity_t` |  |
| `SLMMX_IO_ADDR0` | `SLM_CPU_0_IO_ADDR0` | B |
| `SLMMX_IO_ADDR1` | `SLM_CPU_0_IO_ADDR1` | B |
| `SLMMX_IO_ADDR2` | `SLM_CPU_0_IO_ADDR2` | B |
| `SLMMX_IO_ADDR3` | `SLM_CPU_0_IO_ADDR3` | B |
| `SLMMX_IO_DISC_OUT` | `SLM_CPU_0_IO_DISC_OUT` | B |
| `SLMMX_IO_EN_BUFFN` | `SLM_CPU_0_IO_EN_BUFFN` | B |
| `SLMMX_IO_RESET_N` | `SLM_CPU_0_IO_RESET_N` | B |
| `SLMMX_IO_SPARE` | `SLM_CPU_0_IO_SPARE` |  |
| `SLMMX_IO_SPARE_IN` | `SLM_CPU_0_IO_SPARE_IN` |  |
| `SLMMX_IO_SPI_MISO` | `SLM_CPU_0_IO_SPI_MISO` | B |
| `SLMMX_IO_SPI_MOSI` | `SLM_CPU_0_IO_SPI_MOSI` | B |
| `SLMMX_IO_SPI_NSS` | `SLM_CPU_0_IO_SPI_NSS` | B |
| `SLMMX_IO_SPI_SCK` | `SLM_CPU_0_IO_SPI_SCK` | B |
| `SLMMX_IO_SSELN1` | `SLM_CPU_0_IO_SSELN1` | B |
| `slmmx_ip_pack` | `slm_cpu_0_ip_pack` |  |
| `slmmx_ip_unpack` | `slm_cpu_0_ip_unpack` |  |
| `SLMMX_KEY_ADDR` | `SLM_CPU_0_KEY_ADDR` | B |
| `SLMMX_KEY_SIZE` | `SLM_CPU_0_KEY_SIZE` | B |
| `slmmx_mac_derived` | `slm_cpu_0_mac_derived` |  |
| `slmmx_mac_is_unicast` | `slm_cpu_0_mac_is_unicast` |  |
| `slmmx_mailbox_check` | `slm_cpu_0_mailbox_check` |  |
| `SLMMX_MAILBOX_MAGIC` | `SLM_CPU_0_MAILBOX_MAGIC` |  |
| `SLMMX_MAILBOX_SALT` | `SLM_CPU_0_MAILBOX_SALT` |  |
| `SLMMX_NET_F_DHCP` | `SLM_CPU_0_NET_F_DHCP` | H |
| `slmmx_port` | `slm_cpu_0_port` |  |
| `slmmx_port_flash_` | `slm_cpu_0_port_flash_` |  |
| `slmmx_port_flash_erase_sector` | `slm_cpu_0_port_flash_erase_sector` |  |
| `slmmx_port_flash_program_word` | `slm_cpu_0_port_flash_program_word` |  |
| `SLMMX_RAM_BASE` | `SLM_CPU_0_RAM_BASE` |  |
| `SLMMX_RAM_END` | `SLM_CPU_0_RAM_END` |  |
| `SLMMX_REASON_` | `SLM_CPU_0_REASON_` |  |
| `SLMMX_REASON_FACTORY` | `SLM_CPU_0_REASON_FACTORY` |  |
| `SLMMX_REASON_NONE` | `SLM_CPU_0_REASON_NONE` |  |
| `SLMMX_REASON_UPLOAD` | `SLM_CPU_0_REASON_UPLOAD` | H |
| `slmmx_remember_network` | `slm_cpu_0_remember_network` | H |
| `SLMMX_SERIAL_LEN` | `SLM_CPU_0_SERIAL_LEN` |  |
| `SLMMX_STATIC_ASSERT` | `SLM_CPU_0_STATIC_ASSERT` |  |
| `SLMMX_TFTP_FILE_FIRMWARE` | `SLM_CPU_0_TFTP_FILE_FIRMWARE` |  |
| `SLMMX_TFTP_FILE_IDENTITY` | `SLM_CPU_0_TFTP_FILE_IDENTITY` |  |
| `SLMMX_TFTP_FILE_INFO` | `SLM_CPU_0_TFTP_FILE_INFO` |  |
| `SLMMX_TFTP_MAX_BLKSIZE` | `SLM_CPU_0_TFTP_MAX_BLKSIZE` |  |
| `SLMMX_TFTP_OPT_CRC32` | `SLM_CPU_0_TFTP_OPT_CRC32` |  |
| `SLMMX_TFTP_PORT` | `SLM_CPU_0_TFTP_PORT` |  |
| `SLMMX_TP1` | `SLM_CPU_0_TP1` |  |
| `SLMMX_UID_BASE` | `SLM_CPU_0_UID_BASE` |  |
| `SLMMX_UID_PTR` | `SLM_CPU_0_UID_PTR` |  |

## SLMBackplane symbol coverage

Every `SLMMX_`/`slmmx_` identifier in the SLMBackplane sources maps above: `SLMMX_FLASH_PTR`, `SLMMX_IO_ADDR0`, `SLMMX_IO_ADDR1`, `SLMMX_IO_ADDR2`, `SLMMX_IO_ADDR3`, `SLMMX_IO_DISC_OUT`, `SLMMX_IO_EN_BUFFN`, `SLMMX_IO_RESET_N`, `SLMMX_IO_SPI_MISO`, `SLMMX_IO_SPI_MOSI`, `SLMMX_IO_SPI_NSS`, `SLMMX_IO_SPI_SCK`, `SLMMX_IO_SSELN1`, `SLMMX_KEY_ADDR`, `SLMMX_KEY_SIZE`. Its `#error` text names the `SLMMX_IO_*` pin family, which becomes `SLM_CPU_0_IO_*`.

## HAL-local macros (not provided by the core)

Defined in the FACTS HAL itself; rename them in the HAL with the same prefix rule: `SLMMX_ANALOG` -> `SLM_CPU_0_ANALOG`, `SLMMX_COUNT_OF` -> `SLM_CPU_0_COUNT_OF`, `SLMMX_DISCRETE` -> `SLM_CPU_0_DISCRETE`, `SLMMX_IO_MAPPING_ENTRY` -> `SLM_CPU_0_IO_MAPPING_ENTRY`, `SLMMX_MODULE_CONFIG_BYTES` -> `SLM_CPU_0_MODULE_CONFIG_BYTES`, `SLMMX_MODULE_CONFIG_ENTRY` -> `SLM_CPU_0_MODULE_CONFIG_ENTRY`.

