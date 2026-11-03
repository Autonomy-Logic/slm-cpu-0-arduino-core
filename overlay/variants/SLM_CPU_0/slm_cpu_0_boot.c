/*
 * slm_cpu_0_boot.c - identity, config log and CRC helpers. See boot_shared.h.
 */
#include "boot_shared.h"

#include <string.h>

static const uint32_t crc_nibble[16] = {
    0x00000000u, 0x1DB71064u, 0x3B6E20C8u, 0x26D930ACu,
    0x76DC4190u, 0x6B6B51F4u, 0x4DB26158u, 0x5005713Cu,
    0xEDB88320u, 0xF00F9344u, 0xD6D6A3E8u, 0xCB61B38Cu,
    0x9B64C2B0u, 0x86D3D2D4u, 0xA00AE278u, 0xBDBDF21Cu,
};

uint32_t slm_cpu_0_crc32(uint32_t crc, const void *data, size_t len)
{
    const uint8_t *p = (const uint8_t *)data;
    crc = ~crc;
    while (len--) {
        crc ^= *p++;
        crc = (crc >> 4) ^ crc_nibble[crc & 0x0Fu];
        crc = (crc >> 4) ^ crc_nibble[crc & 0x0Fu];
    }
    return ~crc;
}

static void flash_read(uint32_t addr, void *dst, size_t len)
{
    const volatile uint8_t *src = SLM_CPU_0_FLASH_PTR(addr);
    uint8_t *d = (uint8_t *)dst;
    while (len--) *d++ = *src++;
}

static int flash_is_blank(uint32_t addr, size_t len)
{
    const volatile uint8_t *p = SLM_CPU_0_FLASH_PTR(addr);
    while (len--) {
        if (*p++ != 0xFFu) return 0;
    }
    return 1;
}

/* ---- MAC ------------------------------------------------------------------ */

int slm_cpu_0_mac_is_unicast(const uint8_t mac[6])
{
    static const uint8_t zero[6] = {0};
    if (mac[0] & 0x01u) return 0;   /* multicast / broadcast */
    return memcmp(mac, zero, 6) != 0;
}

/* MAC derived from the unique ID, matching the bootloader's derivation. */
void slm_cpu_0_mac_derived(uint8_t mac[6])
{
    uint32_t uid1 = SLM_CPU_0_UID_PTR[1];
    uint32_t uid2 = SLM_CPU_0_UID_PTR[2];
    mac[0] = 0x00;
    mac[1] = 0x80;
    mac[2] = 0xE1;
    mac[3] = (uint8_t)(uid1 >> 8);
    mac[4] = (uint8_t)(uid2 >> 16);
    mac[5] = (uint8_t)(uid2 >> 8);
}

/* ---- Identity -------------------------------------------------------------- */

int slm_cpu_0_ident_is_valid(const slm_cpu_0_identity_t *rec)
{
    if (rec->magic != SLM_CPU_0_IDENT_MAGIC || rec->version != SLM_CPU_0_IDENT_VERSION) return 0;
    if (slm_cpu_0_crc32(0, rec, offsetof(slm_cpu_0_identity_t, crc)) != rec->crc) return 0;
    return slm_cpu_0_mac_is_unicast(rec->mac);
}

void slm_cpu_0_ident_seal(slm_cpu_0_identity_t *rec)
{
    rec->crc = slm_cpu_0_crc32(0, rec, offsetof(slm_cpu_0_identity_t, crc));
}

const slm_cpu_0_identity_t *slm_cpu_0_ident_newest(void)
{
    const slm_cpu_0_identity_t *best = NULL;
    uint32_t best_seq = 0;
    for (uint32_t i = 0; i < SLM_CPU_0_IDENT_REC_SLOTS; i++) {
        uint32_t addr = SLM_CPU_0_IDENT_REC_BASE + i * SLM_CPU_0_IDENT_REC_SIZE;
        if (flash_is_blank(addr, SLM_CPU_0_IDENT_REC_SIZE)) break;
        slm_cpu_0_identity_t rec;
        flash_read(addr, &rec, sizeof(rec));
        if (!slm_cpu_0_ident_is_valid(&rec)) continue;   /* torn write: skip, keep scanning */
        if (best == NULL || rec.seq >= best_seq) {
            best = (const slm_cpu_0_identity_t *)SLM_CPU_0_FLASH_PTR(addr);
            best_seq = rec.seq;
        }
    }
    return best;
}

int slm_cpu_0_get_mac(uint8_t mac[6])
{
    const slm_cpu_0_identity_t *rec = slm_cpu_0_ident_newest();
    if (rec != NULL) {
        memcpy(mac, (const void *)rec->mac, 6);
        return 1;
    }
    slm_cpu_0_mac_derived(mac);
    return 0;
}

/* ---- Config log ------------------------------------------------------------ */

int slm_cpu_0_cfg_is_valid(const slm_cpu_0_cfg_entry_t *e)
{
    if (e->magic != SLM_CPU_0_CFG_MAGIC) return 0;
    if (e->type == 0 || e->type > SLM_CPU_0_CFG_TYPE_MAX || e->len > SLM_CPU_0_CFG_DATA_SIZE) return 0;
    return slm_cpu_0_crc32(0, e, offsetof(slm_cpu_0_cfg_entry_t, crc)) == e->crc;
}

/* Index of the first blank slot, or SLM_CPU_0_CFG_SLOTS when the log is full. */
static uint32_t cfg_first_free(void)
{
    for (uint32_t i = 0; i < SLM_CPU_0_CFG_SLOTS; i++) {
        if (flash_is_blank(SLM_CPU_0_CFG_BASE + i * SLM_CPU_0_CFG_ENTRY_SIZE, SLM_CPU_0_CFG_ENTRY_SIZE)) return i;
    }
    return SLM_CPU_0_CFG_SLOTS;
}

const slm_cpu_0_cfg_entry_t *slm_cpu_0_cfg_latest(uint16_t type)
{
    const slm_cpu_0_cfg_entry_t *found = NULL;
    uint32_t end = cfg_first_free();
    for (uint32_t i = 0; i < end; i++) {
        uint32_t addr = SLM_CPU_0_CFG_BASE + i * SLM_CPU_0_CFG_ENTRY_SIZE;
        slm_cpu_0_cfg_entry_t e;
        flash_read(addr, &e, sizeof(e));
        if (slm_cpu_0_cfg_is_valid(&e) && e.type == type) found = (const slm_cpu_0_cfg_entry_t *)SLM_CPU_0_FLASH_PTR(addr);
    }
    return found;
}

static int program_block(uint32_t addr, const void *src, size_t len)
{
    const uint8_t *p = (const uint8_t *)src;
    for (size_t off = 0; off < len; off += 4) {
        uint32_t w;
        memcpy(&w, p + off, 4);
        if (slm_cpu_0_port_flash_program_word(addr + (uint32_t)off, w) != 0) return -1;
    }
    const volatile uint8_t *chk = SLM_CPU_0_FLASH_PTR(addr);
    for (size_t i = 0; i < len; i++) {
        if (chk[i] != p[i]) return -1;
    }
    return 0;
}

int slm_cpu_0_cfg_append(uint16_t type, const void *data, uint16_t len)
{
    if (type == 0 || type > SLM_CPU_0_CFG_TYPE_MAX || len > SLM_CPU_0_CFG_DATA_SIZE) return -1;

    slm_cpu_0_cfg_entry_t e;
    memset(&e, 0xFF, sizeof(e));
    e.magic = SLM_CPU_0_CFG_MAGIC;
    e.type = type;
    e.len = len;
    memset(e.data, 0, sizeof(e.data));
    memcpy(e.data, data, len);
    e.crc = slm_cpu_0_crc32(0, &e, offsetof(slm_cpu_0_cfg_entry_t, crc));

    uint32_t slot = cfg_first_free();
    if (slot >= SLM_CPU_0_CFG_SLOTS) {
        /* Full: keep the current entry of every other type, erase, rewrite. */
        slm_cpu_0_cfg_entry_t keep[SLM_CPU_0_CFG_TYPE_MAX];
        uint32_t nkeep = 0;
        for (uint16_t t = 1; t <= SLM_CPU_0_CFG_TYPE_MAX; t++) {
            if (t == type) continue;
            const slm_cpu_0_cfg_entry_t *cur = slm_cpu_0_cfg_latest(t);
            if (cur != NULL) memcpy(&keep[nkeep++], (const void *)cur, sizeof(slm_cpu_0_cfg_entry_t));
        }
        if (slm_cpu_0_port_flash_erase_sector(SLM_CPU_0_CFG_SECTOR) != 0) return -1;
        slot = 0;
        for (uint32_t i = 0; i < nkeep; i++, slot++) {
            if (program_block(SLM_CPU_0_CFG_BASE + slot * SLM_CPU_0_CFG_ENTRY_SIZE, &keep[i], sizeof(keep[i])) != 0) return -1;
        }
    }
    return program_block(SLM_CPU_0_CFG_BASE + slot * SLM_CPU_0_CFG_ENTRY_SIZE, &e, sizeof(e));
}
