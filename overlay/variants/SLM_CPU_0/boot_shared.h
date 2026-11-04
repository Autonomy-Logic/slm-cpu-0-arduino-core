/*
 * boot_shared.h - flash layout and hand-off formats shared with the network
 * bootloader. Little-endian. Names may change; values, layouts and wire
 * strings must match the bootloader (docs/bootloader-contract.md).
 */
#ifndef SLM_CPU_0_BOOT_SHARED_H
#define SLM_CPU_0_BOOT_SHARED_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#define SLM_CPU_0_STATIC_ASSERT(cond, msg) static_assert(cond, msg)
#else
#define SLM_CPU_0_STATIC_ASSERT(cond, msg) _Static_assert(cond, msg)
#endif

/* Flash map (STM32F407VE, 512 KB): 0x08000000 sectors 0-1 bootloader,
 * 0x08008000 sector 2 identity (backplane key, MAC records; bootloader-only
 * writes), 0x0800C000 sector 3 config log, 0x08010000 sectors 4-7 application. */
#define SLM_CPU_0_FLASH_BASE            0x08000000u
#define SLM_CPU_0_FLASH_END             0x08080000u

#define SLM_CPU_0_BOOT_BASE             0x08000000u
#define SLM_CPU_0_BOOT_SIZE             0x00008000u

#define SLM_CPU_0_IDENT_BASE            0x08008000u
#define SLM_CPU_0_IDENT_SIZE            0x00004000u
#define SLM_CPU_0_IDENT_SECTOR          2u
#define SLM_CPU_0_KEY_ADDR              SLM_CPU_0_IDENT_BASE
#define SLM_CPU_0_KEY_SIZE              32u
#define SLM_CPU_0_IDENT_REC_BASE        (SLM_CPU_0_IDENT_BASE + 0x100u)
#define SLM_CPU_0_IDENT_REC_SIZE        64u
#define SLM_CPU_0_IDENT_REC_SLOTS       ((SLM_CPU_0_IDENT_SIZE - 0x100u) / SLM_CPU_0_IDENT_REC_SIZE)

#define SLM_CPU_0_CFG_BASE              0x0800C000u
#define SLM_CPU_0_CFG_SIZE              0x00004000u
#define SLM_CPU_0_CFG_SECTOR            3u

#define SLM_CPU_0_APP_BASE              0x08010000u
#define SLM_CPU_0_APP_END               SLM_CPU_0_FLASH_END
#define SLM_CPU_0_APP_MAX_SIZE          (SLM_CPU_0_APP_END - SLM_CPU_0_APP_BASE)
#define SLM_CPU_0_APP_FIRST_SECTOR      4u
#define SLM_CPU_0_APP_LAST_SECTOR       7u
/* Leading image bytes the bootloader holds in RAM and programs last, after
 * the whole image is verified. */
#define SLM_CPU_0_APP_COMMIT_SIZE       1024u

#define SLM_CPU_0_RAM_BASE              0x20000000u
#define SLM_CPU_0_RAM_END               0x20020000u

#define SLM_CPU_0_UID_BASE              0x1FFF7A10u

/* Flash is read through this macro so host tests can back it with an array. */
#ifndef SLM_CPU_0_FLASH_PTR
#define SLM_CPU_0_FLASH_PTR(addr)       ((const volatile uint8_t *)(uintptr_t)(addr))
#endif
#ifndef SLM_CPU_0_UID_PTR
#define SLM_CPU_0_UID_PTR               ((const volatile uint32_t *)(uintptr_t)SLM_CPU_0_UID_BASE)
#endif

/* Sector number for a flash address, or -1 outside the 512 KB array. */
static inline int slm_cpu_0_flash_sector_of(uint32_t addr)
{
    if (addr < SLM_CPU_0_FLASH_BASE || addr >= SLM_CPU_0_FLASH_END) return -1;
    uint32_t off = addr - SLM_CPU_0_FLASH_BASE;
    if (off < 0x10000u) return (int)(off / 0x4000u);
    if (off < 0x20000u) return 4;
    return 5 + (int)((off - 0x20000u) / 0x20000u);
}

static inline uint32_t slm_cpu_0_flash_sector_base(int sector)
{
    if (sector < 4) return SLM_CPU_0_FLASH_BASE + (uint32_t)sector * 0x4000u;
    if (sector == 4) return SLM_CPU_0_FLASH_BASE + 0x10000u;
    return SLM_CPU_0_FLASH_BASE + 0x20000u + (uint32_t)(sector - 5) * 0x20000u;
}

/* ---- Identity record (sector 2, from SLM_CPU_0_IDENT_REC_BASE) -------------
 * Append-only; the valid record with the highest `seq` wins. Written only by
 * the bootloader (identity.bin over TFTP), which assigns seq and the CRC. */
#define SLM_CPU_0_IDENT_MAGIC           0x544E4449u   /* "IDNT" */
#define SLM_CPU_0_IDENT_VERSION         1u
#define SLM_CPU_0_SERIAL_LEN            20u

typedef struct __attribute__((packed)) {
    uint32_t magic;
    uint16_t version;
    uint16_t hw_rev;
    uint32_t seq;
    uint8_t  mac[6];
    uint8_t  flags;
    uint8_t  reserved0;
    char     serial[SLM_CPU_0_SERIAL_LEN];   /* ASCII, NUL padded */
    uint8_t  ip[4];                      /* factory default address */
    uint8_t  mask[4];
    uint8_t  gw[4];
    uint8_t  reserved1[8];
    uint32_t crc;                        /* slm_cpu_0_crc32 over bytes [0, 60) */
} slm_cpu_0_identity_t;

SLM_CPU_0_STATIC_ASSERT(sizeof(slm_cpu_0_identity_t) == SLM_CPU_0_IDENT_REC_SIZE, "identity record must be 64 bytes");

/* ---- Config log (sector 3) -------------------------------------------------
 * Append-only 32-byte entries; the last valid entry of each type is current.
 * When full, the writer keeps the current entry per type, erases, rewrites. */
#define SLM_CPU_0_CFG_MAGIC             0x47464343u   /* "CCFG" */
#define SLM_CPU_0_CFG_ENTRY_SIZE        32u
#define SLM_CPU_0_CFG_DATA_SIZE         20u
#define SLM_CPU_0_CFG_SLOTS             (SLM_CPU_0_CFG_SIZE / SLM_CPU_0_CFG_ENTRY_SIZE)

enum {
    SLM_CPU_0_CFG_NET = 1,   /* slm_cpu_0_cfg_net_t: last network settings the app used */
    SLM_CPU_0_CFG_APP = 2,   /* slm_cpu_0_cfg_app_t: image the bootloader last committed */
};
#define SLM_CPU_0_CFG_TYPE_MAX          2u

typedef struct __attribute__((packed)) {
    uint32_t magic;
    uint16_t type;
    uint16_t len;
    uint8_t  data[SLM_CPU_0_CFG_DATA_SIZE];
    uint32_t crc;                        /* slm_cpu_0_crc32 over bytes [0, 28) */
} slm_cpu_0_cfg_entry_t;

SLM_CPU_0_STATIC_ASSERT(sizeof(slm_cpu_0_cfg_entry_t) == SLM_CPU_0_CFG_ENTRY_SIZE, "config entry must be 32 bytes");

#define SLM_CPU_0_NET_F_DHCP            0x01u

typedef struct __attribute__((packed)) {
    uint8_t ip[4];
    uint8_t mask[4];
    uint8_t gw[4];
    uint8_t dns[4];
    uint8_t flags;
    uint8_t reserved[3];
} slm_cpu_0_cfg_net_t;

typedef struct __attribute__((packed)) {
    uint32_t size;       /* image length in bytes */
    uint32_t crc32;      /* slm_cpu_0_crc32 of the image, as sent by the upload tool */
    uint32_t hwcrc;      /* STM32 CRC unit over the image padded to 4 bytes with 0xFF */
    uint32_t sp;         /* first two vector table words, so an image written */
    uint32_t reset;      /* some other way (SWD) is recognised as not this one */
} slm_cpu_0_cfg_app_t;

SLM_CPU_0_STATIC_ASSERT(sizeof(slm_cpu_0_cfg_net_t) == SLM_CPU_0_CFG_DATA_SIZE, "net entry must fill the data field");
SLM_CPU_0_STATIC_ASSERT(sizeof(slm_cpu_0_cfg_app_t) == SLM_CPU_0_CFG_DATA_SIZE, "app entry must fill the data field");

/* ---- App -> bootloader mailbox (RTC backup registers) ---------------------
 * Survives a system reset: the app writes the request and its address, then
 * resets; the bootloader consumes it. */
#define SLM_CPU_0_BKP_MAGIC             0u
#define SLM_CPU_0_BKP_BOOTCOUNT         1u
#define SLM_CPU_0_BKP_REASON            2u
#define SLM_CPU_0_BKP_IP                3u
#define SLM_CPU_0_BKP_MASK              4u
#define SLM_CPU_0_BKP_GW                5u
#define SLM_CPU_0_BKP_CHECK             6u

#define SLM_CPU_0_MAILBOX_MAGIC         0x534C4D42u   /* "BMLS": stay in the bootloader */
#define SLM_CPU_0_MAILBOX_SALT          0xA5C35A3Cu

enum {
    SLM_CPU_0_REASON_NONE = 0,
    SLM_CPU_0_REASON_UPLOAD = 1,     /* editor download, FC 0x4C */
    SLM_CPU_0_REASON_FACTORY = 2,
};

/* Uncleared consecutive watchdog resets after which the bootloader waits for
 * a new image. The app clears the count after SLM_CPU_0_APP_STABLE_MS. */
#define SLM_CPU_0_CRASH_LOOP_LIMIT      3u
#define SLM_CPU_0_APP_STABLE_MS         30000u

static inline uint32_t slm_cpu_0_ip_pack(const uint8_t ip[4])
{
    return (uint32_t)ip[0] | ((uint32_t)ip[1] << 8) | ((uint32_t)ip[2] << 16) | ((uint32_t)ip[3] << 24);
}

static inline void slm_cpu_0_ip_unpack(uint32_t v, uint8_t ip[4])
{
    ip[0] = (uint8_t)v;
    ip[1] = (uint8_t)(v >> 8);
    ip[2] = (uint8_t)(v >> 16);
    ip[3] = (uint8_t)(v >> 24);
}

static inline uint32_t slm_cpu_0_mailbox_check(uint32_t reason, uint32_t ip, uint32_t mask, uint32_t gw)
{
    return SLM_CPU_0_MAILBOX_MAGIC ^ reason ^ ip ^ mask ^ gw ^ SLM_CPU_0_MAILBOX_SALT;
}

/* ---- Bootloader network service -------------------------------------------- */
#define SLM_CPU_0_TFTP_PORT             69u
#define SLM_CPU_0_TFTP_FILE_FIRMWARE    "firmware.bin"
#define SLM_CPU_0_TFTP_FILE_IDENTITY    "identity.bin"
#define SLM_CPU_0_TFTP_FILE_INFO        "info"
#define SLM_CPU_0_TFTP_OPT_CRC32        "crc32"      /* 8 hex digits, slm_cpu_0_crc32 of the file */
#define SLM_CPU_0_TFTP_MAX_BLKSIZE      1468u        /* fills a 1500-byte MTU */

#define SLM_CPU_0_DEFAULT_IP            {169, 254, 10, 10}
#define SLM_CPU_0_DEFAULT_MASK          {255, 255, 0, 0}

/* ---- Shared helpers (slm_cpu_0_boot.c) ----------------------------------------- */

/* Standard CRC-32 (zlib/Ethernet). Start with crc = 0. */
uint32_t slm_cpu_0_crc32(uint32_t crc, const void *data, size_t len);

int  slm_cpu_0_mac_is_unicast(const uint8_t mac[6]);
void slm_cpu_0_mac_derived(uint8_t mac[6]);

int  slm_cpu_0_ident_is_valid(const slm_cpu_0_identity_t *rec);
void slm_cpu_0_ident_seal(slm_cpu_0_identity_t *rec);
/* Newest valid identity record in flash, or NULL. */
const slm_cpu_0_identity_t *slm_cpu_0_ident_newest(void);
/* MAC from the newest identity record, else the unique-ID derived one.
 * Returns 1 when it came from an identity record. */
int  slm_cpu_0_get_mac(uint8_t mac[6]);

int  slm_cpu_0_cfg_is_valid(const slm_cpu_0_cfg_entry_t *e);
const slm_cpu_0_cfg_entry_t *slm_cpu_0_cfg_latest(uint16_t type);
/* Append an entry, compacting the sector first when it is full.
 * Returns 0 on success. Needs the slm_cpu_0_port_flash_* functions. */
int  slm_cpu_0_cfg_append(uint16_t type, const void *data, uint16_t len);

/* ---- Flash access supplied by each target --------------------------------- */
int  slm_cpu_0_port_flash_erase_sector(uint32_t sector);
int  slm_cpu_0_port_flash_program_word(uint32_t addr, uint32_t value);

#ifdef __cplusplus
}
#endif

#endif /* SLM_CPU_0_BOOT_SHARED_H */
